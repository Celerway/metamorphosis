# Metamorphosis

A simple MQTT -> Kafka bridge and concentrator. Note that we'll use 
[Red Panda](https://github.com/vectorizedio/redpanda) instead of Kafka. It is simpler (no zookeeper) 
and faster (no JVM), and claims protocol compatibility with Kafka. You should consider it.

This is a protocol bridge between MQTT and Kafka. It'll connect to a broker, subscription and listen for messages. When
a message is received, we'll give it to kafka. It is meant to be running in a k8s pod.

If Kafka is unavailable we'll try to spool the messages to memory, so they can be recovered. If we can't write to Kafka,
we'll retry every 10 seconds. Once we reconnect, we dump all the messages we have.

Once Kafka and MQTT are connected, Metamorphosis will listen on `HEALTH_PORT` (cleartext http) and deliver metrics if a
client requests `/metrics`. We'll also answer /healthz, so you can have k8s poll this url.

Note that you need to make sure that the topic exists in Red Panda / Kafka or that auto creation of topics is enabled.

Note that there are limited guarantees given. We try to keep messages ordered. However, during restart, k8s will start a
new instance of the daemon before the old one is shut down. During this short period you'll see messages duplicates.

Also note that the bridge will issue messages in order to test that it can talk to Kafka. These will be given the MQTT
topic "test". Ignore these messages in your consumer.

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

## Design

Three main packages

* bridge glues together mqtt and kafka. If we ever want to do transformations, it happens here.
* mqtt contains the mqtt stuff
* kafka for the kafka stuff

In addition, there is an observability package which deals with prometheus stuff and responds to k8s health checks.

Six goroutines should be running at any point in time:

* one is listening to MQTT
* one is talking to Kafka
* one is moving messages between these two. We could have sent the messages directly, but the overhead is small, and we
  want to have the opportunity to transform messages. So this layering makes sense.
* one is serving HTTP, so we have some observability and health.
* one is listening for observability events on the observability channel and updates the prom counters
* one silly little one is just listening for SIGTERM and SIGINT

## Key dependencies

 * [go-kafka](https://github.com/segmentio/kafka-go), a nice native Go Kafka client.
 * [paho.mqtt.golang](https://github.com/eclipse/paho.mqtt.golang), MQTT client. Doesn't support MQTT 5.
 * [logrus](https://github.com/sirupsen/logrus), our preferred logger

## Performance

For us the most important thing is reliability. So we do synchronous writes which block the writer. 
This is pretty slow, but we're sure not to lose any messages. If you need more performance you can do the following:
 * Have more kafka workers. They will load balance the channel.
 * Increase the batch timeout. This will make to writer block some more but, it'll batch the writes, lessening the load 
   on Kafka. Not sure how well this works across multiple goroutines, though.

### Todo: Tls against Kafka

We don't need this ourselves, but PRs are welcome. Should be too hard. #goodfirsttask

### Todo: testing

Testing will happen something like this.

- generate some TLS certs and keys
- spin up a minimal broker (mosquitto?)
- spin up the bridge with a mock Kafka backend
- issues some messages to the broker
- restart the broker and see that it reconnect
- inject some errors into the mocked kafka backend
- verify that the bridge does what it is supposed to do

### Todo: Validation of messages

Not sure if we need this as the messages are validated when they are consumed from Red Panda.

Depending on configuration the bridge will either reject the message or warn of the message doesn't pass validation.

### Todo: Support for multiple subscriptions.

Perhaps this could be done as simply as setting MQTT_TOPIC to several strings separated by , og ; or similar. We don't
need this ourselves, but PRs are welcome. Should be too hard. #goodfirsttask

