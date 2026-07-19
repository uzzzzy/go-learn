.PHONY: fmt lint

fmt:
	gofmt -w .

lint:
	golangci-lint run

format: fmt lint