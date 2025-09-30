.PHONY: build run lint test clean

all: build

build:
        @echo "Building the project..."
        $(eval VERSION := $(shell toml get --toml-path .\namor.toml version))
        $(eval COMMIT_SHA := $(shell git rev-parse --short HEAD))
        go build -ldflags "-X 'main.Env=development' -X 'main.Version=$(VERSION)' -X 'main.Commit=$(COMMIT_SHA)'" -o build/namor main.go
        @echo "Build completed."

run: build
        @echo "Running the project..."
        ./build/namor
        @echo "Run completed."

lint:
        @echo "Linting the code..."
        golangci-lint run ./...
        @echo "Linting completed."

image_%:
        @make -f docker.mk $* --no-print-directory --silent

clean:
        @echo "Cleaning up..."
        rm -r build/
        @echo "Cleanup completed."