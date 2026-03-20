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
	@echo "Test $$(go test -cover ./gopty/ | grep -oE 'coverage: [0-9.]+%')"
	@echo ""
	@echo "Lines of code (excluding comments and blanks):"
	@total=0; \
	for f in gopty/*.go cmd/main.go; do \
		n=$$(grep -cvE '^\s*(//|$$)' "$$f"); \
		total=$$((total + n)); \
		printf "%8d %s\n" "$$n" "$$f"; \
	done; \
	printf "%8d total\n" "$$total"

test:
	go test -race ./gopty/
