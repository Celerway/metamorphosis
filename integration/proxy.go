package main

// Simple proxy between MQTT/Kafka and metamorphosis
// Used for integration testing.

import (
	"flag"
	"fmt"
	"io"
	"net"
)

func main() {
	const mqttPort = 1883
	const kafkaPort = 9092
	var (
		myMqttPort  int = mqttPort + 10000
		myKafkaPort int = kafkaPort + 10000
	)

	flag.IntVar(&myMqttPort, "mqtt-port", myMqttPort, "MQTT port")
	flag.IntVar(&myKafkaPort, "kafka-port", myKafkaPort, "Kafka port")
	flag.Parse()
	go proxy(myMqttPort, mqttPort)
	go proxy(myKafkaPort, kafkaPort)
	select {} // block forever.
}

func proxy(src, dst int) {

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", src))
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("error accepting connection", err)
			continue
		}
		go func() {
			conn2, err := net.Dial("tcp", fmt.Sprintf(":%d", dst))
			if err != nil {
				fmt.Println("error dialing remote addr", err)
				return
			}
			go io.Copy(conn2, conn)
			io.Copy(conn, conn2)
			conn2.Close()
			conn.Close()
		}()
	}

}
