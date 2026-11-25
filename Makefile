VERSION ?= v1.2.0
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build-linux build-macos build-windows build-all test

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o build/linux/hop ./cmd/hop

build-macos:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o build/macos/hop-amd64 ./cmd/hop
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o build/macos/hop-arm64 ./cmd/hop

build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o build/windows/hop.exe ./cmd/hop

build-all: build-linux build-macos build-windows

test:
	go test -v ./...
