.DEFAULT_GOAL := build

GOTOOLS := golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow \
           honnef.co/go/tools/cmd/staticcheck

.PHONY: tools
tools:
	@echo "Installing tools..."
	@for tool in $(GOTOOLS); do \
		go install $$tool@latest; \
	done

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint: fmt
	staticcheck ./...

.PHONY: vet
vet: fmt
	go vet ./...
	shadow ./...

.PHONY: test
test: vet
	go test -v ./...

.PHONY: build
build: test
	go mod tidy
	@mkdir -p bin
	go build -o bin/batch ./cmd/batch
	go build -o bin/api   ./cmd/api

.PHONY: clean
clean:
	rm -rf bin/