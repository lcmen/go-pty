BINARY = go-pty

.PHONY: build test clean

build:
	go build -o $(BINARY) ./cmd

test:
	go test ./gopty/

clean:
	rm -f $(BINARY)
