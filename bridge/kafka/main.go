package kafka

import log "github.com/sirupsen/logrus"

func Run() {
	log.WithFields(log.Fields{
		"module": "bridge/kafka",
	})

}
