# Metamorphosis

A simple MQTT -> Kafka bridge and concentrator.
(Note that we'll use Red Panda instead of Kafka. It is simpler and faster, and claims protocol compatibility 
with Kafka). You should consider it.

This is a protocol bridge between MQTT and Kafka. It'll connect to a broker, subscription and listen
for messages. When a message is received, we'll give it to kafka.

If Kafka is unavailable we'll try to spool the messages to local storage, so they can be recovered. 
Not sure about the details here, but I suspect we can dump streams of JSON in some format that 
can be easily ingested into Kafka / Red Panda.

Metamorphosis will listen on PROM_PORT (cleartext http) and deliver metrics if a client 
requests `/metrics`.

Note that you need to make sure that the topic exists in Red Panda / Kafka or that auto creation
of topics is enabled.

## Message format

Each message that is written to Kafka will look like this:
```
type Message struct {
  Topic   string   // The topic of the originating MQTT message.
  Content []byte   // base64 encoded as we don't know anything about what it contains.
}
```
So, then reading from Kafka we'll need to look at the topic and call the relevant handler for that
type of message. We don't really know what is inside the actual message we get from MQTT, so the 
content of the message is base64 encoded.

## Design

Three main packages
 * bridge glues together mqtt and kafka. If we ever want to do transformations, it happens here.
 * mqtt contains the mqtt stuff
 * kafka for the kafka stuff

In addition, there is an observability package which deals with prometheus stuff.

Six goroutines should be running at any point in time:
 * one is listening to MQTT
 * one is talking to Kafka
 * one is moving messages between these two. We could have sent the messages directly, but the overhead is
   small, and we want to have the opportunity to transform messages. So this layering makes sense.
 * one is serving HTTP, so we have some observability and health.
 * one is listening for observability events on the observability channel and updates the prom counters
 * one silly little one is just listening for SIGTERM and SIGINT

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

Depending on configuration the bridge will either
reject the message or warn of the message doesn't pass validation.

### Todo: Support for multiple subscriptions.

Perhaps this could be done as simply as setting MQTT_TOPIC to several strings separated 
by , og ; or similar.


