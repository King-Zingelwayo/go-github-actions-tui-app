.PHONY: build run clean deps test

# Build the application
build:
	go build -o bin/indlovu ./cmd

# Run the application
run: build
	./bin/indlovu

# Install dependencies
deps:
	go mod tidy
	go mod download

# Clean build artifacts
clean:
	rm -rf bin/

# Run tests
test:
	go test ./...

# Install for system-wide use
install: build
	sudo cp bin/indlovu /usr/local/bin/

# Development setup
dev-setup: deps
	go install github.com/cosmtrek/air@latest

# Live reload for development
dev:
	air

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o bin/indlovu-linux-amd64 ./cmd
	GOOS=darwin GOARCH=amd64 go build -o bin/indlovu-darwin-amd64 ./cmd
	GOOS=windows GOARCH=amd64 go build -o bin/indlovu-windows-amd64.exe ./cmd