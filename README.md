# Metamorphosis

A simple MQTT -> Kafka bridge and concentrator. Note that we'll use 
[Red Panda](https://github.com/vectorizedio/redpanda) instead of Kafka. It is simpler (no zookeeper) 
and faster (no JVM), and claims protocol compatibility with Kafka. You should consider it. Note that Metamorphosis 
uses packages written for Kafka, so it is likely to work just as well with Kafka.

This is a protocol bridge between MQTT and Kafka. It'll connect to a broker, using HOSTNAME as the client ID, 
subscription and listen for messages. When a message is received it is placed in a buffer and written to Kafka 
either based on message volume or given an interval. 

Upon startup it can, if configured to do so, issue a message to Kafka. This can be very useful as any misconfiguration 
will then be immediately detected. If this initial message can't be sent Metamorphosis will exit with an error.

When up and running Metamorphosis will listen for messages on the configured MQTT topic and write them to Kafka. Note 
that what we get from MQTT is binary data so the messages written to Kafka will be base64 encoded. 

If Kafka becomes unavailable we'll try to spool the messages to memory, so they can be recovered. It will retry every 10 
seconds. Once we reconnect, it'll dump all the messages we have buffered.

Metamorphosis will listen on `HEALTH_PORT` (cleartext http) and deliver metrics if a
client requests `/metrics`. We'll also answer /healthz, so you can have k8s poll this url.

Metamorphosis will auto-create Kafka topics.

Note that there are limited guarantees given. If you run Metamorphosis in a k8s deployment you can potentially see 
message duplication. If you, on the other hand, run Metamorphosis as a StatefulSet you might lose messages upon restart.

## Message format

Each message that is written to Kafka will look like this:

```
type Message struct {
  Topic   string   // The topic of the originating MQTT message.
  Content []byte   // base64 encoded as we don't know anything about what it contains.
}
```

So, then reading from Kafka we'll need to look at the topic and call the relevant handler for that type of message. We
don't really know what is inside the actual message we get from MQTT, so the content of the message is base64 encoded.

## Development

You'll need an .env file to run this locally or command line options. I recommend having a ssh port forward
to a Kafka server.

Suggested `.env` file:
```
LOG_LEVEL=trace
ROOT_CA=.tls/ca.pem
MQTT_CLIENT_CERT=.tls/client.pem
MQTT_CLIENT_KEY=.tls/client.key

MQTT_BROKER=localhost
MQTT_TOPIC="test/#"

KAFKA_BROKER=localhost
KAFKA_PORT=9092
KAFKA_TOPIC="mqtt"
```

I use [standard-version](https://www.npmjs.com/package/standard-version) to maintain the changelog and tags.

## Design

Three main packages

* bridge glues together mqtt and kafka. If we ever want to do transformations, it happens here.
* mqtt contains the mqtt stuff
* kafka for the kafka stuff. this is the only one containing any meaningful logic.

In addition, there is an observability package which deals with prometheus stuff and responds to k8s health checks.

## Key dependencies

 * [go-kafka](https://github.com/segmentio/kafka-go), a nice native Go Kafka client.
 * [paho.mqtt.golang](https://github.com/eclipse/paho.mqtt.golang), MQTT client. Doesn't support MQTT 5.
 * [logrus](https://github.com/sirupsen/logrus), our preferred logger

## Performance

For us the most important thing is reliability. So we do synchronous writes which block the writer. 
This is pretty slow, but we're sure not to lose any messages. If you need more performance you can increase the
batch size.


### Todo: Tls against Kafka

We don't need this ourselves, but PRs are welcome. Should be too hard. #goodfirsttask

### Todo: Support for multiple subscriptions.

Perhaps this could be done as simply as setting MQTT_TOPIC to several strings separated by , og ; or similar. We don't
need this ourselves, but PRs are welcome. Should be too hard. #goodfirsttask

### Things we're not really interested in adding.

* If you need to transform the messages, I would encourage you to look at Red Pandas WASM transformations. 
  I don't see the need of adding this to the bridge. Feel free to fork if you need this.
* Message validation. Red Panda supports this.