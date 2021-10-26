package integration

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

const mosquittoCmd = "mosquitto"
const rpkCmd = "rpk"
const redPandaContainer = "docker.vectorized.io/vectorized/redpanda:latest"

func startMqtt(ctx context.Context) {
	path, err := exec.LookPath(mosquittoCmd)
	if err != nil {
		fmt.Println("Can't find mosquitto in path " + err.Error())
		panic(err)
	}

	fmt.Printf("Starting %s (%s)\n", mosquittoCmd, path)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		cmd := exec.CommandContext(ctx, path, "-d")
		err := cmd.Run()
		if err != nil {
			output, err := cmd.CombinedOutput()
			fmt.Printf("startMqtt: %s\n", output)
			panic("Error:" + err.Error())
		}
		wg.Done()
		fmt.Println("Running mosquitto until context is cancelled")
		<-ctx.Done()
		output, err := exec.Command("pkill", mosquittoCmd).CombinedOutput()
		if err != nil {
			fmt.Println("pkill failed", string(output))
			panic(err)
		} else {
			fmt.Println("pkill mosquitto OK")
		}
	}()
	wg.Wait()
	fmt.Println("MQTT started")
}

// Do a quick and dirty restart of MQTT
func restartMqtt() {
	output, err := exec.Command("pkill", mosquittoCmd).CombinedOutput()
	if err != nil {
		fmt.Println("pkill failed", string(output))
		panic(err)
	}
	time.Sleep(100 * time.Millisecond)

	path, err := exec.LookPath(mosquittoCmd)
	if err != nil {
		fmt.Println("Can't find mosquitto in path " + err.Error())
		panic(err)
	}

	cmd := exec.Command(path, "-d")
	err = cmd.Run()
	if err != nil {
		output, err := cmd.CombinedOutput()
		fmt.Printf("startMqtt: %s\n", output)
		panic("Error:" + err.Error())
	}
}
func startKafka(ctx context.Context, wg *sync.WaitGroup) {
	bctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	out, err := cli.ImagePull(bctx, redPandaContainer, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(os.Stdout, out)
	if err != nil {
		panic("Couldn't copy to Stdout: " + err.Error())
	}

	hostBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: "9092",
	}
	containerPort, err := nat.NewPort("tcp", "9092")
	if err != nil {
		panic("Unable to get the port")
	}
	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}
	cmdLine := strslice.StrSlice{"redpanda", "start", "--overprovisioned", "--smp", "1", "--memory", "256M", "--reserve-memory", "0M", "--node-id", "0", "--check=false"}
	fmt.Println("Starting red panda container")
	resp, err := cli.ContainerCreate(bctx,
		&container.Config{
			Image: redPandaContainer,
			Cmd:   cmdLine,
		},
		&container.HostConfig{
			PortBindings: portBinding,
		},
		nil, nil, "redpanda0")
	if err != nil {
		fmt.Println("ContainerCreate() failed - panic()")
		panic(err)
	}
	fmt.Printf("Red panda container with ID '%s' started\n", resp.ID)
	if err := cli.ContainerStart(bctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		fmt.Println("ContainerStart failed - panic()")
		panic(err)
	}
	go func() {
		fmt.Println("Waiting for context cancel")
		<-ctx.Done()
		fmt.Println("Context shutting down. Stopping red panda")
		timeout := time.Second
		err := cli.ContainerStop(bctx, resp.ID, &timeout)
		if err != nil {
			fmt.Println("Error stopping container")
			panic(err)
		}
		fmt.Println("Red panda container stopped")
		err = cli.ContainerRemove(bctx, resp.ID, types.ContainerRemoveOptions{})
		if err != nil {
			fmt.Println("Error removing container")
			panic(err)
		}
		fmt.Println("Red panda container removed")
		wg.Done()
	}()
}

func waitForKafka() {
	for {
		cmd := exec.Command(rpkCmd, "topic", "list")
		err := cmd.Run()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	fmt.Println("Red Panda running")
}

func kafkaTopic(t *testing.T, action, topic string) {
	cmd := exec.Command(rpkCmd, "topic", action, topic)
	var stdout, stderr bytes.Buffer
	log.Infof("kafka topic(%s): %s", action, topic)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Errorf("Kafka topic (%s) \nstdout: %s \nstderr: %s err: %s", action, stdout.String(), stderr.String(), err)
	}
	if action == "create" {
		time.Sleep(3 * time.Second) // Just sleep a bit to make sure that kafka catches up.
	}
	log.Debug("kafka topic executed")
	log.Debugf("Stdout: %s, \nStderr: %s", stdout.String(), stderr.String())
}
