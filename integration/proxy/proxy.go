package proxy

// Simple proxy between MQTT/Kafka and metamorphosis
// Used for integration testing.

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
)

func StartProxy(ctx context.Context, myMqttPort, myKafkaPort int, logger *log.Entry) {
	const mqttPort = 1883
	const kafkaPort = 9092
	go proxy(myMqttPort, mqttPort, logger)
	go proxy(myKafkaPort, kafkaPort, logger)
	select {
	case <-ctx.Done():
	}
}

func proxy(src, dst int, logger *log.Entry) {
	logger.Infof("Setting up a proxy from port %d --> %d", src, dst)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", src))
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("error accepting connection", err)
			continue
		}
		go func() {
			conn2, err := net.Dial("tcp", fmt.Sprintf(":%d", dst))
			if err != nil {
				logger.Error("error dialing remote addr", err)
				return
			}
			go io.Copy(conn2, conn)
			io.Copy(conn, conn2)
			conn2.Close()
			conn.Close()
		}()
	}

}
