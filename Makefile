build:
	go build -o bin/diamonds cmd/main.go

test:
	go test ./...