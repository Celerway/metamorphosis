build:
	go build -o bin/metamorphosis cmd/main.go cmd/options.go

test:
	go test ./...