package mqtt

import (
	"context"
	"fmt"
	"github.com/celerway/metamorphosis/bridge/observability"
	paho "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

func Run(ctx context.Context, p MqttParams) {
	log.Debugf("Starting MQTT Worker.")
	log.Debugf("Broker: %s:%d (tls: %v)", p.Broker, p.Port, p.Tls)
	c := mqttClient{
		broker:     p.Broker,
		port:       p.Port,
		topic:      p.Topic,
		clientId:   "metamorphosis", // Todo: Figure out a proper ID
		tls:        p.Tls,
		ch:         p.Channel,
		waitGroup:  p.WaitGroup,
		obsChannel: p.ObsChannel,
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
	c.paho = client
	log.Debug("Spinning off the MQTT worker")
	go c.mainloop(ctx)
}

// mainloop
// This is a goroutine. When you return it dies.
// Connects to the MQTT broker, subscribes and processes messages.
// All the works happens in the event handler.
func (client mqttClient) mainloop(ctx context.Context) {
	log.Debug("In mqtt main loop")
	client.waitGroup.Add(1)

	client.handleConnect()
	client.subscribe() // Also sets up handlers.

	// Here we start blocking the goroutine and wait for shutdown.
	// If we need to keep track of something we can wrap this in a loop
	// Just be sure not to introduce any races.
	select {
	case <-ctx.Done():
		log.Debug("MQTT client shutting down.")
		// Todo: There seems to be a race condition in the Paho Mqtt library that
		// we trigger here from time to time.
		client.unsubscribe()
		client.paho.Disconnect(100)
		log.Debug("MQTT disconnected")
	}

	client.waitGroup.Done()
	log.Info("MQTT client exiting")
}

// handleConnect
// Connects to the broker. Blocks until the connection is established.
func (client *mqttClient) handleConnect() {
	log.Debug("MQTT client starting connect to broker.")
	token := client.paho.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Could not connect to broker: %s", token.Error())
		// Todo: wtf do we do here?
		// Do we retry? Fail fatally and assume that k8s will restart the container?
	}
	log.Infof("Worker '%v' connected to MQTT %s:%d", client.clientId, client.broker, client.port)
}

func (client *mqttClient) handleDisconnect(pahoClient paho.Client, err error) {
	log.Errorf("handleDisconnect invoked with error: %s", err)
	log.Info("Reconnecting to broker.")
	client.handleConnect()

}

func (client mqttClient) unsubscribe() {
	token := client.paho.Unsubscribe(client.topic)
	token.Wait()
	log.Infof("Unsubscribed from topic '%s'", client.topic)
}
func (client mqttClient) subscribe() {
	token := client.paho.Subscribe(client.topic, 1, client.messageHandler)
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
	client.obsChannel <- observability.MqttRecieved
}
