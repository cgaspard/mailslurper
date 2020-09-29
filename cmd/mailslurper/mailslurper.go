// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

//go:generate esc -o ./www/www.go -pkg www -ignore DS_Store|README\.md|LICENSE|www\.go -prefix /www/ ./www

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo"
	"github.com/mailslurper/mailslurper/pkg/mailslurper"
	"github.com/mailslurper/mailslurper/pkg/ui"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

const (
	// Version of the MailSlurper Server application
	SERVER_VERSION string = "1.14.1"

	// Set to true while developing
	DEBUG_ASSETS bool = false
)

var config *mailslurper.Configuration
var database mailslurper.IStorage
var logger *logrus.Entry
var renderer *ui.TemplateRenderer
var mailItemChannel chan *mailslurper.MailItem
var smtpListenerContext context.Context
var smtpListenerCancel context.CancelFunc
var smtpListener *mailslurper.SMTPListener
var connectionManager *mailslurper.ConnectionManager
var cacheService *cache.Cache

var admin *echo.Echo
var service *echo.Echo

type Environment struct {
	LogFormat  string
	LogLevel   string
	ConfigPath string
}

var env Environment

func main() {
	var err error

	parseArgs()

	logger = mailslurper.GetLogger(env.LogLevel, env.LogFormat, "MailSlurper")
	logger.Infof("Starting MailSlurper Server v%s", SERVER_VERSION)

	renderer = ui.NewTemplateRenderer(DEBUG_ASSETS)
	setupConfig(env.ConfigPath)

	if err = config.Validate(); err != nil {
		logger.WithError(err).Fatalf("Invalid configuration")
	}

	cacheService = cache.New(time.Minute*time.Duration(config.AuthTimeoutInMinutes), time.Minute*time.Duration(config.AuthTimeoutInMinutes))

	setupDatabase()
	setupSMTP()
	setupAdminListener()
	setupServicesListener()

	defer database.Disconnect()

	if config.AutoStartBrowser {
		ui.StartBrowser(config, logger)
	}

	/*
	 * Block this thread until we get an interrupt signal. Once we have that
	 * start shutting everything down
	 */
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)

	<-quit

	ctx, cancel := context.WithTimeout(smtpListenerContext, 20*time.Second)
	defer cancel()

	smtpListenerCancel()

	if err = admin.Shutdown(ctx); err != nil {
		logger.Fatalf("Error shutting down admin listener: %s", err.Error())
	}

	if err = service.Shutdown(ctx); err != nil {
		logger.Fatalf("Error shutting down service listener: %s", err.Error())
	}
}

func parseArgs() {

	paramLogFormat := "logformat"
	paramLogLevel := "loglevel"
	paramConfigPath := "configpath"

	envLogFormat := os.Getenv(paramLogFormat)
	envLogLevel := os.Getenv(paramLogLevel)
	envConfigPath := os.Getenv(paramConfigPath)

	flag.StringVar(&env.LogFormat, paramLogFormat, "simple", "Format for logging. 'simple' or 'json'.")
	flag.StringVar(&env.LogLevel, paramLogLevel, "info", "Level of logs to write. Valid values are 'debug', 'info', or 'error'.")
	flag.StringVar(&env.ConfigPath, paramConfigPath, "config.json", "Path to config.json")

	flag.Parse()

	if len(envLogFormat) > 0 {
		env.LogFormat = envLogFormat
	}

	if len(envLogLevel) > 0 {
		env.LogLevel = envLogLevel
	}

	if len(envConfigPath) > 0 {
		env.ConfigPath = envConfigPath
	}

}
