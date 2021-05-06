package mqtt

import (
	"fmt"
	paho "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

func Run(p MqttParams) {
	log.Debugf("Starting MQTT Worker.")
	log.Debugf("Broker: %s:%d (tls: %v)", p.Broker, p.Port, p.Tls)
	c := mqttClient{
		broker:    p.Broker,
		port:      p.Port,
		topic:     p.Topic,
		clientId:  "metamorphosis", // Todo: Figure out a proper ID
		tls:       p.Tls,
		ch:        p.Channel,
		waitGroup: p.WaitGroup,
		ctx:       p.Ctx,
	}
	opts := paho.NewClientOptions()
	if p.Tls {
		opts.SetTLSConfig(p.TlsConfig)
		opts.AddBroker(fmt.Sprintf("ssl://%s:%d", p.Broker, p.Port))
		c.tls = true
	} else {
		opts.AddBroker(fmt.Sprintf("mqtt://%s:%d", p.Broker, p.Port))
	}
	opts.SetClientID(c.clientId)
	opts.SetConnectionLostHandler(c.handleDisconnect)
	client := paho.NewClient(opts)
	c.client = client
	log.Debug("Spinning off the MQTT worker")
	c.waitGroup.Add(1)
	go c.mainloop()
}

// mainloop
// This is a goroutine. When you return it dies.
// Connects to the MQTT broker, subscribes and processes messages.
func (client mqttClient) mainloop() {
	log.Debug("In mqtt main loop")

	client.handleConnect()
	client.subscribe()

	keepRunning := true
	for keepRunning {
		select {
		case <-client.ctx.Done():
			log.Debug("MQTT bridge shutting down.")
			keepRunning = false
		}
	}
	client.waitGroup.Done()
	log.Info("MQTT bridge exiting")
}

// handleConnect
// Connects to the broker. Blocks until the connection is established.
func (c *mqttClient) handleConnect() {
	log.Debug("Worker starting connect to broker.")
	token := c.client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Could not connect to broker: %s", token.Error())
		// Todo: wtf do we do here?
		// Do we retry? Fail fatally and assume that k8s will restart the container?
	}
	log.Infof("Worker '%v' connected", c.clientId)
}

func (c *mqttClient) handleDisconnect(pahoClient paho.Client, err error) {
	log.Errorf("handleDisconnect invoked with error: %s", err)
	log.Info("Reconnecting")
	c.handleConnect()

}

func (client mqttClient) subscribe() {
	token := client.client.Subscribe(client.topic, 1, client.messageHandler)
	token.Wait()
	log.Infof("Subscribed to topic '%s'", client.topic)
}

func (client mqttClient) messageHandler(pahoClient paho.Client, msg paho.Message) {
	log.Debugf("Got message on topic %s. Message: %s", msg.Topic(), string(msg.Payload()))
	chMsg := MqttChannelMessage{
		Topic:   msg.Topic(),
		Content: msg.Payload(),
	}
	client.ch <- chMsg
}
