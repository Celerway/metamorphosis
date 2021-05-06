package kafka

import (
	"context"
	"crypto/tls"
	"sync"
)

type KafkaChannelMessage struct {
	Topic   string
	Content []byte
}

type MessageChannel chan KafkaChannelMessage

type KafkaParams struct {
	Server    string
	Port      int
	Clientid  string
	Tls       bool
	TlsConfig *tls.Config
	Ctx       context.Context
	Channel   MessageChannel
	WaitGroup *sync.WaitGroup
	Topic     string
}

type kafkaClient struct {
	tlsConfig *tls.Config
	server    string
	port      int
	clientId  string
	tls       bool
	ch        MessageChannel
	ctx       context.Context
	waitGroup *sync.WaitGroup
	topic     string
}
