.DEFAULT_GOAL := build

GOTOOLS := golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow \
           honnef.co/go/tools/cmd/staticcheck

.PHONY: tools openapi_bundle fmt vet lint test build

tools:
	@echo "Installing Go tools..."
	@for tool in $(GOTOOLS); do \
		go install $$tool@latest; \
	done
	@echo "Checking swagger-cli..."
	@if ! command -v swagger-cli >/dev/null 2>&1; then \
		echo "Installing swagger-cli via npm..."; \
		npm install -g @apidevtools/swagger-cli; \
	else \
		echo "swagger-cli already installed."; \
	fi

openapi_bundle:
	@echo "Bundling OpenAPI spec..."
	swagger-cli bundle internal/server/openapi/openapi.yaml --dereference -o internal/server/openapi/openapi.bundle.yaml

fmt:
	go fmt ./...

vet:
	go vet ./...
	shadow ./...

lint:
	staticcheck ./...

test:
	GO_ENV=test go test -v ./...

build: tools openapi_bundle fmt vet lint test 
	go mod tidy
	@mkdir -p bin
	go build -o bin/batch ./cmd/batch
	go build -o bin/api   ./cmd/api
