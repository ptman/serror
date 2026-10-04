.PHONY: test test-race bench lint coverage coverage-html clean

# Run all tests
test:
	go test -v ./...

# Run tests with race detector
test-race:
	go test -race -v ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Run golangci-lint
lint:
	golangci-lint run ./...

# Generate coverage profile and print per-function coverage
coverage:
	go test -coverprofile=coverage.out -covermode=count ./...
	@go tool cover -func=coverage.out

# Generate HTML coverage report to file
coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html

# Remove generated files
clean:
	rm -f coverage.out coverage.html
