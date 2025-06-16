.DEFAULT_GOAL := build

GOTOOLS := golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow \
           honnef.co/go/tools/cmd/staticcheck

.PHONY: tools openapi_bundle fmt vet lint test build compile

tools:
	@echo "Installing Go tools..."
	@for tool in $(GOTOOLS); do \
		go install $$tool@latest; \
	done
	@echo "Checking redocly CLI..."
	@if ! command -v redocly >/dev/null 2>&1; then \
		echo "Installing @redocly/cli via npm..."; \
		npm install -g @redocly/cli; \
	else \
		echo "redocly already installed."; \
	fi

openapi_bundle:
	@echo "Bundling OpenAPI spec using redocly..."
	redocly bundle internal/server/openapi/openapi.yaml \
	  --output internal/server/openapi/openapi.bundle.yaml

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

compile:
	protoc api/v1/*.proto \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.
