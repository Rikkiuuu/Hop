VERSION ?= v1.0.0
VER_NUM := $(patsubst v%,%,$(VERSION))
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
 go build -ldflags="$(LDFLAGS)" -o build/hop ./cmd/hop

build-linux:
 mkdir -p build/linux
 GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o build/linux/hop ./cmd/hop

build-macos:
 mkdir -p build/macos/amd64 build/macos/arm64
 GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o build/macos/amd64/hop ./cmd/hop
 GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o build/macos/arm64/hop ./cmd/hop

build-windows:
 mkdir -p build/windows
 GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o build/windows/hop.exe ./cmd/hop

build-all: build-linux build-macos build-windows

release: build-all
 mkdir -p build/release
 tar -czf build/release/hop_$(VER_NUM)_linux_amd64.tar.gz -C build/linux hop
 tar -czf build/release/hop_$(VER_NUM)_darwin_amd64.tar.gz -C build/macos/amd64 hop
 tar -czf build/release/hop_$(VER_NUM)_darwin_arm64.tar.gz -C build/macos/arm64 hop
 zip -j build/release/hop_$(VER_NUM)_windows_amd64.zip build/windows/hop.exe

test:
 go test -v ./...

clean:
 rm -rf build/hop build/linux/hop build/macos/amd64 build/macos/arm64 build/windows/hop.exe build/release
