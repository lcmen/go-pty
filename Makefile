.PHONY: build clean install lint test

BINARY = go-pty

build: clean
	go build -o $(BINARY) ./cmd

clean:
	rm -f $(BINARY)

lint:
	go vet ./...
	go fmt ./...

install: build
	cp $(BINARY) $(HOME)/.local/bin

test:
	go test -race ./gopty/
