# All Go commands run inside backend/ (module root).
GOLANGCI := golangci-lint

.PHONY: build vet test lint ci clean

build:
	cd backend && go build ./...

vet:
	cd backend && go vet ./...

test:
	cd backend && go test ./...

lint:
	@if command -v $(GOLANGCI) >/dev/null; then cd backend && $(GOLANGCI) run ./...; else echo "golangci-lint not installed, running go vet instead:"; cd backend && go vet ./...; fi

ci: build vet test

clean:
	cd backend && go clean ./...
