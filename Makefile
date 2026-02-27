BINARY = go-pty

.PHONY: build test lint clean

build:
	go build -o $(BINARY) ./cmd

test:
	go test -race ./gopty/

lint:
	go vet ./...
	go fmt ./...

clean:
	rm -f $(BINARY)
