package bridge

import (
	"github.com/celerway/metamorphosis/bridge/kafka"
	"github.com/celerway/metamorphosis/bridge/mqtt"
	log "github.com/sirupsen/logrus"
)

// Here I put the stuff that glues the mqtt to the kafka.
// Not sure if this should be a separate package. Let's keep things simple atm.

func (br bridge) run() {
	keepRunning := true
	br.wg.Add(1)
	for keepRunning {
		select {
		case chMsg := <-br.mqttCh:
			br.glueMsgHandler(chMsg)
		case <-br.ctx.Done():
			log.Debug("Glue shutting down.")
			keepRunning = false
			break
		}
	}
	br.wg.Done()
}

func (br bridge) glueMsgHandler(msg mqtt.MqttChannelMessage) {
	kafkaMsg := kafka.KafkaChannelMessage{
		Topic:   msg.Topic,
		Content: msg.Content,
	}
	br.kafkaCh <- kafkaMsg
}
