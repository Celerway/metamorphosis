package mqtt

import (
	"context"
	"fmt"
	"github.com/celerway/metamorphosis/bridge/observability"
	paho "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
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
	client.connect()   // blocks and aborts on failure.
	client.subscribe() //  blocks. Also sets up handlers.
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
	client.logger.Debug("MQTT client starting connect to broker.")
	token := client.paho.Connect()
	if token.Wait() && token.Error() != nil {
		client.logger.Fatalf("Could not connect to broker: %s", token.Error())
		// What do we do here? My suggestion is to fail the service (abort)
		// and assume that k8s will restart the pod.
	}
	client.logger.Infof("Worker '%v' connected to MQTT %s:%d", client.clientId, client.broker, client.port)
}
func (client mqttClient) handleConnect(_ paho.Client) {
	client.logger.Info("Connection to MQTT broker established")
}

func (client *mqttClient) handleDisconnect(_ paho.Client, err error) {
	client.logger.Errorf("handleDisconnect invoked with error: %s", err)
	client.logger.Info("Reconnecting to broker.")
	client.connect()
}

func (client mqttClient) unsubscribe() {
	token := client.paho.Unsubscribe(client.topic)
	token.Wait()
	client.logger.Infof("Unsubscribed from topic '%s'", client.topic)
}
func (client mqttClient) subscribe() {
	token := client.paho.Subscribe(client.topic, 1, client.messageHandler)
	token.Wait()
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
