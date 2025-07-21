# Set your Go binary name here
BINARY_NAME=myapp

# Go environment (can override in command line)
GOPROXY_URL=https://proxy.golang.org,direct

# Online build
.PHONY: build
build:
	@echo ">> Building with GOPROXY=$(GOPROXY_URL)..."
	GOPROXY=$(GOPROXY_URL) go mod tidy
	GOPROXY=$(GOPROXY_URL) go build -o $(BINARY_NAME) .

# Run the app
.PHONY: run
run: build
	@echo ">> Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Build using vendored dependencies (offline)
.PHONY: build-vendor
build-vendor:
	@echo ">> Building with vendored modules..."
	go build -mod=vendor -o $(BINARY_NAME) .

# Vendor all dependencies
.PHONY: vendor
vendor:
	@echo ">> Tidying and vendoring dependencies..."
	GOPROXY=$(GOPROXY_URL) go mod tidy
	GOPROXY=$(GOPROXY_URL) go mod vendor

# Clean build artifacts
.PHONY: clean
clean:
	@echo ">> Cleaning up..."
	rm -f $(BINARY_NAME)
