package main

import (
	"github.com/TrollEyeSecurity/ccsyslogingester/config"
	"github.com/TrollEyeSecurity/ccsyslogingester/service"
	"github.com/getsentry/sentry-go"
	"log"
	"time"
)

func main() {
	appConfiguration := config.LoadConfiguration("/etc/ccsyslog/config.json")
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
	defer sentry.Flush(2 * time.Second)
	service.ShipperService()
}
