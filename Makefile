VERSION := 0.1.0
BINARY := credential-vault
LDFLAGS := -X credential-vault/internal/cli.Version=$(VERSION)

# OS/Arch targets
LINUX_AMD64 := $(BINARY)-linux-amd64
LINUX_ARM64 := $(BINARY)-linux-arm64
DARWIN_AMD64 := $(BINARY)-darwin-amd64
DARWIN_ARM64 := $(BINARY)-darwin-arm64
WINDOWS_AMD64 := $(BINARY)-windows-amd64.exe
WINDOWS_ARM64 := $(BINARY)-windows-arm64.exe

.PHONY: build build-all clean test vet run release

# Build for current platform
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/credential-vault

# Build for all supported platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(LINUX_AMD64) ./cmd/credential-vault
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(LINUX_ARM64) ./cmd/credential-vault
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DARWIN_AMD64) ./cmd/credential-vault
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DARWIN_ARM64) ./cmd/credential-vault
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(WINDOWS_AMD64) ./cmd/credential-vault
	GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(WINDOWS_ARM64) ./cmd/credential-vault

# Run tests
test:
	go test -v ./tests/... -timeout 30s

# Run go vet
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -f $(BINARY) $(BINARY)-* $(BINARY).exe

# Run locally
run: build
	./$(BINARY) help

# Create release archives
release: build-all
	mkdir -p dist
	tar -czf dist/$(BINARY)-v$(VERSION)-linux-amd64.tar.gz $(LINUX_AMD64)
	tar -czf dist/$(BINARY)-v$(VERSION)-linux-arm64.tar.gz $(LINUX_ARM64)
	tar -czf dist/$(BINARY)-v$(VERSION)-darwin-amd64.tar.gz $(DARWIN_AMD64)
	tar -czf dist/$(BINARY)-v$(VERSION)-darwin-arm64.tar.gz $(DARWIN_ARM64)
	zip -q dist/$(BINARY)-v$(VERSION)-windows-amd64.zip $(WINDOWS_AMD64)
	zip -q dist/$(BINARY)-v$(VERSION)-windows-arm64.zip $(WINDOWS_ARM64)
	rm -f $(LINUX_AMD64) $(LINUX_ARM64) $(DARWIN_AMD64) $(DARWIN_ARM64) $(WINDOWS_AMD64) $(WINDOWS_ARM64)
	@echo "Release archives created in dist/"
	@ls -lh dist/