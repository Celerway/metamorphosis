# Metamorphosis

A simple MQTT -> Kafka bridge and concentrator.
(Note that we'll use Red Panda instead of Kafka. It is simpler and faster, but claims 100% protocol compatibility 
with Kafka)

This is a protocol bridge between MQTT and Kafka. It'll connect to a broker, subscription and listen
for messages. One a message is received we'll give it to kafka.

If Kafka is unavailable we'll try to spool the messages to local storage, so they can be recovered. 
Not sure about the details here, but I suspect we can dump streams of JSON in some format that 
can be easily ingested into Kafka / Red Panda.

## Message format

Each message that is written to Kafka will look like this:
```
type MqttChannelMessage struct {
  Topic   string
  Content []byte
}
```
So, then reading from Kafka we'll need to look at the topic and call the relevant handler for that
type of message.

## Design

Four goroutines should be running at any point in time:
 * one is listening to MQTT
 * one is talking to Kafka
 * one is moving messages between these two
 * one is serving HTTP, so we have some observability


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

Depending on configuration the bridge will either
reject the message or warn of the message doesn't pass validation.

### Todo: Support for multiple subscriptions.

Perhaps this could be done as simply as setting MQTT_TOPIC to several strings separated 
by , og ; or similar.


