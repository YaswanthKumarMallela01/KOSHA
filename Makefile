.PHONY: build test lint run clean

BINARY=kosha

build:
	go build -o $(BINARY) ./cmd/kosha

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; fi
	@if command -v gofmt >/dev/null 2>&1; then gofmt -l .; fi

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)

install:
	go install ./cmd/kosha
