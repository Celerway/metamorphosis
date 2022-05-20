package bridge

import (
	kafka "github.com/celerway/metamorphosis/bridge/kafka"
	"github.com/celerway/metamorphosis/bridge/mqtt"
)

// Here I put the stuff that glues the mqtt to the kafka.
// Not sure if this should be a separate package. Let's keep things simple atm.

// Note that this code doesn't used contexts or waitgroups.
// When we exit there is no cleanup to be done.
func (br bridge) mainloop() {
	for msg := range br.mqttCh {
		br.glueMsgHandler(msg)
	}
}

func (br bridge) glueMsgHandler(msg mqtt.ChannelMessage) {
	kafkaMsg := kafka.Message{
		Topic:   msg.Topic,
		Content: msg.Content,
	}
	br.logger.Trace("bridge pushed a message to kafka")
	br.kafkaCh <- kafkaMsg
}
