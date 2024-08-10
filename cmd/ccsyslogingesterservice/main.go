package main

import (
	"fmt"
	"github.com/TrollEyeSecurity/ccsyslogingester/config"
	"github.com/TrollEyeSecurity/ccsyslogingester/service"
	"github.com/getsentry/sentry-go"
	"log"
	"time"
)

func main() {
	appConfiguration, appConfigurationErr := config.LoadConfiguration("/etc/ccsyslog/config.json")
	if appConfigurationErr != nil {
		err := fmt.Errorf("app-configuration error %v", appConfigurationErr)
		log.Println(err)
		return
	}
	if appConfiguration.SentryIoDsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              appConfiguration.SentryIoDsn,
			TracesSampleRate: 1.0,
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
		defer sentry.Flush(2 * time.Second)
	}
	service.IngestService()
}
