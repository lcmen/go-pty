.PHONY: build clean install lint stats test

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

stats:
	@sh stats.sh

test:
	go test -race ./gopty/
