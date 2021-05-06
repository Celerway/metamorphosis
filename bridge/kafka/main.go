package kafka

import (
	"time"
)

func Run(params KafkaParams) {
	params.WaitGroup.Add(1)
	time.Sleep(5 * time.Second)
	params.WaitGroup.Done()
}
