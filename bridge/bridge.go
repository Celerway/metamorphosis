package bridge

import (
	"github.com/celerway/metamorphosis/bridge/kafka"
	"github.com/celerway/metamorphosis/bridge/mqtt"
)

// Here I put the stuff that glues the mqtt to the kafka.
// Not sure if this should be a separate package. Let's keep things simple atm.

func (br bridge) run() {
	go br.mainloop()
}

// Note that this code doesn't used contexts or waitgroups.
// When we exit there is no cleanup to be done.
func (br bridge) mainloop() {
	for msg := range br.mqttCh {
		br.glueMsgHandler(msg)
	}
}

func (br bridge) glueMsgHandler(msg mqtt.MqttChannelMessage) {
	kafkaMsg := kafka.KafkaMessage{
		Topic:   msg.Topic,
		Content: msg.Content,
	}
	br.kafkaCh <- kafkaMsg
}
