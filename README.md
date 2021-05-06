# Metamorphosis

A simple MQTT -> Kafka bridge.
(Note that we'll use Red Panda instead of Kafka. It is simpler and faster, but claims 100% protocol compatibility with Kafka)

This software is currently proprietary. It is just a lot simpler to write this for our specific needs than to make it
generic. This is mainly due to two reason:

* The types. We'll use native Go types for the schema and not having to have any Yaml defining the schema simplifies the
  code a lot.

This is a protocol bridge between MQTT and Kafka. It'll connect to a broker, issue a number of subscriptions and listen
for messages. One a message is received we'll transform it a bit and give it to kafka.

If Kafka is unavailable we'll try to spool the messages to local storage, so they can be recovered. Not sure about the
details here, but I suspect we can dump streams of JSON in some format that can be easily ingested
into Kafka / Red Panda.

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

Upon startup we connect to MQTT and Kafka. Each spinning up a goroutine.  We spin of a b
ridge goroutine which is responsible to push data between the sources.

Both Kafka and MQTT will communicate to the bridge with a channel.

## Validation of messages

Both MQTT and Kafka will validate the messages they send and receive. Depending on configuration the bridge will either
reject the message or warn of the message doesn't pass validation.

## Transforming data

As a message is transferred to the bridge, and the bridge will transmit the message.

In our specific case we'll take some data from the topic of the MQTT message and inject this into the message that is
issued on the kafka topic.





