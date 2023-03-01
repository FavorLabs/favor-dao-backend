.PHONY: all build conf run test clean fmt pre-commit help

TARGET = favorDao
ifeq ($(OS),Windows_NT)
TARGET := $(TARGET).exe
endif

ifeq (n$(CGO_ENABLED),n)
CGO_ENABLED := 1
endif


BUILD_VERSION := $(shell git describe --tags | cut -f 1 -d "-")
BUILD_DATE := $(shell date +'%Y-%m-%d %H:%M:%S')
SHA_SHORT := $(shell git rev-parse --short HEAD)

TAGS = ""
MOD_NAME = favor-dao-backend
LDFLAGS = -X "${MOD_NAME}/pkg/debug.version=${BUILD_VERSION}" \
          -X "${MOD_NAME}/pkg/debug.buildDate=${BUILD_DATE}" \
		  -X "${MOD_NAME}/pkg/debug.commitID=${SHA_SHORT}" -w -s

all: fmt build conf

conf: conf
	[ -f "dist/config.yaml" ] || cp -rf config.yaml dist/

build:
	@go mod download
	@echo Build favorDao
	@go build -trimpath -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o dist/$(TARGET)

run:
	@go run -trimpath -gcflags "all=-N -l" -tags '$(TAGS)' -ldflags '$(LDFLAGS)' .

.PHONY: linux-amd64
linux-amd64:
	@echo Build favorDao [linux-amd64] CGO_ENABLED=$(CGO_ENABLED)
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 go build -trimpath -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o dist/$(TARGET)

.PHONY: darwin-amd64
darwin-amd64:
	@echo Build favorDao [darwin-amd64] CGO_ENABLED=$(CGO_ENABLED)
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 go build -trimpath  -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o dist/$(TARGET)

.PHONY: darwin-arm64
darwin-arm64:
	@echo Build favorDao [darwin-arm64] CGO_ENABLED=$(CGO_ENABLED)
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=arm64 go build -trimpath -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o dist/$(TARGET)

.PHONY: windows-x64
windows-x64:
	@echo Build favorDao [windows-x64] CGO_ENABLED=$(CGO_ENABLED)
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 go build -trimpath  -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o dist/$(TARGET)

clean:
	@go clean
	@find ./dist -type f -exec rm -r {} +

fmt:
	@echo Formatting...
	@go fmt ./internal/...
	@go fmt ./pkg/...
	@go vet -composites=false ./internal/...
	@go vet -composites=false ./pkg/...

test:
	@go test ./...

pre-commit: fmt
	go mod tidy

help:
	@echo "make: make"
	@echo "make run: start api server"
	@echo "make build: build executable"