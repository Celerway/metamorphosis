build:
	go build -o bin/metamorphosis cmd/main.go

test:
	go test ./...