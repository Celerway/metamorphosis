# Metamorphosis

A simple MQTT -> Kafka bridge.

This software is currently proprietary. It is just a lot simpler to write this 
for our specific needs than to make it generic. This is mainly due to two reason:
 * The types. We'll use native Go types for the schema and not having to have 
   any Yaml defining the schema simplifies the code a lot.
 * Transformations. There are no easy ways to plug arbitrary Go code for
   doing transformations.
   
Both of these could be addressed using Go plugins, which would allow us to load user-supplied 
code to do these operations.

This is a protocol bridge between MQTT and Kafka. It'll connect
to a broker, issue a number of subscriptions and listen for 
messages.

## Design

Upon startup we connect to MQTT and Kafka. Each spinning up a 
goroutine.

We spin of a bridge goroutine which is responsible to push data
between the sources.

Both Kafka and MQTT will communicate to the bridge with a channel.


## Validation of messages

Both MQTT and Kafka will validate the messages they send and recieve. Depending on
configuration the bridge will either reject the message or warn of the message doesn't 
pass validation.

## Transforming data

As a message is transfered to the bridge the bridge will transform 
the message.

In our specific case we'll take some data from the topic of the 
MQTT message and inject this into the message that is issued on the kafka topic.





