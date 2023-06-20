package logger

import log "github.com/sirupsen/logrus"

func SetLogger(logLevel log.Level) {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})
	log.SetLevel(logLevel)
}
