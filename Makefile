.PHONY: build
build:
	@echo "Building ./atlassian-mcp-bin"
	@go build -o atlassian-mcp-bin ./cmd/main.go
	@echo "Done."

.PHONY: test
test:
	@go test -race -count=1 ./...

.PHONY: setup-env
setup-env:
	@if [ ! -f .env ]; then \
		cp .env.example .env && echo "Created .env from .env.example"; \
	fi
	@chmod 600 .env && echo ".env permissions set to 600"
