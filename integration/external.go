package integration

import (
	"bytes"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"testing"
	"time"
)

func stopAllServices() {
	exec.Command("pkill", "mosquitto").Run()
	exec.Command("sudo", "rpk", "redpanda", "stop").Run()
}

func startMqtt(ctx context.Context) {
	fmt.Println("Starting MQTT")
	go func() {
		output, err := exec.CommandContext(ctx, "/usr/sbin/mosquitto").CombinedOutput()
		if err != nil {
			fmt.Println(output)
			panic("Error:" + err.Error())
		} else {
			fmt.Println("MQTT started")
		}
	}()
}

func startKafka(ctx context.Context) {
	fmt.Println("Starting Red Panda")

	go func() {
		output, err := exec.CommandContext(ctx, "sudo", "/usr/bin/rpk", "redpanda", "start").CombinedOutput()
		if err != nil {
			fmt.Println(output)
			panic("Error:" + err.Error())
		} else {
			fmt.Println("Kafka started")
		}
	}()
}

func kafkaTopic(t *testing.T, action, topic string) {
	cmd := exec.Command("rpk", "topic", "-v", action, topic)
	var stdout, stderr bytes.Buffer
	log.Infof("kafka topic(%s): %s", action, topic)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Errorf("Kafka topic (%s) stdout: %s stderr: %s err: %s", action, stdout.String(), stderr.String(), err)
	}
	if action == "create" {
		time.Sleep(3 * time.Second) // Just sleep a bit to make sure that kafka catches up.
	}
	log.Debug("kafka topic executed")
	log.Debugf("Stdout: %s, \nStderr: %s", stdout.String(), stderr.String())
}
