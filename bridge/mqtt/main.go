package mqtt

import (
	"context"
	"fmt"
	"github.com/celerway/metamorphosis/bridge/observability"
	paho "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

func Run(ctx context.Context, params MqttParams) {
	log.Debugf("Starting MQTT Worker.")
	log.Debugf("Broker: %s:%d (tls: %v)", params.Broker, params.Port, params.Tls)
	client := mqttClient{
		broker:     params.Broker,
		port:       params.Port,
		topic:      params.Topic,
		clientId:   params.Clientid,
		tls:        params.Tls,
		ch:         params.Channel,
		waitGroup:  params.WaitGroup,
		obsChannel: params.ObsChannel,
		logger:     log.WithFields(log.Fields{"module": "mqtt"}),
	}

	opts := paho.NewClientOptions()
	if params.Tls {
		opts.SetTLSConfig(params.TlsConfig)
		opts.AddBroker(fmt.Sprintf("ssl://%s:%d", params.Broker, params.Port))
		client.tls = true
	} else {
		opts.AddBroker(fmt.Sprintf("mqtt://%s:%d", params.Broker, params.Port))
	}
	opts.SetClientID(client.clientId)
	opts.SetConnectionLostHandler(client.handleDisconnect)
	opts.SetOnConnectHandler(client.handleConnect)
	pahoClient := paho.NewClient(opts)
	client.paho = pahoClient
	client.connect() // blocks and aborts on failure.
	client.logger.Info("Starting MQTT client worker")
	go client.mainloop(ctx)
}

// mainloop
// This is a goroutine. When you return it dies.
// Connects to the MQTT broker, subscribes and processes messages.
// All the works happens in the event handler.
func (client mqttClient) mainloop(ctx context.Context) {
	client.waitGroup.Add(1)
	// Here we start blocking the goroutine and wait for shutdown.
	// If we need to keep track of something we can wrap this in a loop
	select {
	case <-ctx.Done():
		client.logger.Info("MQTT client context is cancelled. Shutting down.")
		client.unsubscribe()
		time.Sleep(100 * time.Millisecond)
		client.paho.Disconnect(100)
		client.logger.Debug("MQTT disconnected")
	}

	client.waitGroup.Done()
	client.logger.Info("MQTT client exiting")
}

// connect
// Connects to the broker. Blocks until the connection is established.
func (client *mqttClient) connect() {
	const connectionAttempts = 10
	connected := false
	attempts := 0
	for !connected && attempts < connectionAttempts {
		client.logger.Infof("MQTT client connection attempt %d: connect to broker %s:%d (tls: %v).",
			attempts, client.broker, client.port, client.tls)
		token := client.paho.Connect()
		if token.Wait() && token.Error() != nil {
			client.logger.Errorf("Could not connect to MQTT (%s). ", token.Error())
			time.Sleep(100 * time.Millisecond) // Add some time so the broker isn't rushed by reconnects.
		} else {
			connected = true
			client.subscribe() //  blocks. Also sets up handlers.
		}
		attempts++
	}
	if !connected && attempts >= connectionAttempts {
		client.logger.Errorf("Max number of connection attempt reached. Giving up and aborting.")
		time.Sleep(time.Second)
		os.Exit(1)
	}
	client.logger.Infof("Worker '%v' connected to MQTT %s:%d", client.clientId, client.broker, client.port)
}
func (client mqttClient) handleConnect(_ paho.Client) {
	client.logger.Info("Connection to MQTT broker established")
}

func (client *mqttClient) handleDisconnect(_ paho.Client, err error) {
	client.logger.Errorf("handleDisconnect invoked with error: %s", err)
	client.logger.Info("Reconnecting to broker.")
}

func (client mqttClient) unsubscribe() {
	token := client.paho.Unsubscribe(client.topic)
	if token.Wait() && token.Error() != nil {
		client.logger.Errorf("Could not unsubscribe from %s:  %s", client.topic, token.Error())
	} else {
		client.logger.Infof("Unsubscribed from topic '%s'", client.topic)
	}
}
func (client mqttClient) subscribe() {
	success := false
	for !success {
		token := client.paho.Subscribe(client.topic, 1, client.messageHandler)
		if token.Wait() && token.Error() != nil {
			client.logger.Errorf("Could not subscribe to '%s':  %s", client.topic, token.Error())
			time.Sleep(100 * time.Millisecond) // Sleep some between attempts
		} else {
			success = true
		}
	}
	client.logger.Infof("Subscribed to topic '%s'", client.topic)
}

func (client mqttClient) messageHandler(_ paho.Client, msg paho.Message) {
	client.logger.Tracef("Got message on topic %s. Message: %s", msg.Topic(), string(msg.Payload()))
	chMsg := MqttChannelMessage{
		Topic:   msg.Topic(),
		Content: msg.Payload(),
	}
	client.ch <- chMsg
	client.obsChannel <- observability.MqttRecieved
}
