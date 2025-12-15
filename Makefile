.PHONY: build run validate monitor roast all test clean help

# Default target
help:
	@echo "GitHub Project Agent - Makefile Commands"
	@echo ""
	@echo "Build:"
	@echo "  make build          - Build the binary"
	@echo ""
	@echo "Run:"
	@echo "  make validate       - Validate all open issues"
	@echo "  make monitor        - Check for stale tasks (once)"
	@echo "  make monitor-daemon - Run monitor as daemon"
	@echo "  make roast          - Generate product roast and suggestions"
	@echo "  make all            - Run all agent tasks"
	@echo ""
	@echo "Development:"
	@echo "  make test           - Run tests"
	@echo "  make clean          - Clean build artifacts"
	@echo ""
	@echo "Note: Make sure to set environment variables (see .env.example)"

build:
	@echo "Building..."
	@go build -o bin/github-project-agent main.go
	@echo "Build complete: bin/github-project-agent"

run: build
	@./bin/github-project-agent $(ARGS)

validate:
	@go run main.go -mode=validate

validate-issue:
	@if [ -z "$(ISSUE)" ]; then \
		echo "Usage: make validate-issue ISSUE=123"; \
		exit 1; \
	fi
	@go run main.go -mode=validate -issue=$(ISSUE)

monitor:
	@go run main.go -mode=monitor -once

monitor-daemon:
	@go run main.go -mode=monitor -daemon

roast:
	@go run main.go -mode=roast

all:
	@go run main.go -mode=all

test:
	@go test ./...

clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean
	@echo "Clean complete"

