package integration

type Message struct {
	Id           int
	RandomString string
	Checksum     [32]byte
}

type KMessage struct {
	Topic   string
	Content []byte
}
