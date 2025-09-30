IMAGE_NAME := namor
# 	VOLUME_NAME ?= namor-shared-volume
# 	SHARED_VOLUME_PATH := /namor-shared-volume
CONFIG_FILE ?= ./config.yaml
SECRET ?= supersecret

.PHONY: build
build:
        @echo "======================================"
        @echo "Starting Docker build for image: $(IMAGE_NAME)"
        @echo "Build context: $(shell pwd)"
        @echo "--------------------------------------"
        docker build -t $(IMAGE_NAME):dev .
        @if [ $$? -eq 0 ]; then \
                echo "Docker image '$(IMAGE_NAME)' built successfully."; \
        else \
                echo "Docker build failed for image '$(IMAGE_NAME)'."; \
                exit 1; \
        fi
        @echo "======================================"

# Run the Docker container
.PHONY: run
run:
        docker run --rm \
                --user root \
                --name $(IMAGE_NAME)-container \
                -v $(VOLUME_NAME):$(SHARED_VOLUME_PATH) \
                -v /var/run/docker.sock:/var/run/docker.sock:rw \
                -e DOCKER_HOST=unix:///var/run/docker.sock \
                $(IMAGE_NAME):dev \
                --config-file=$(CONFIG_FILE)
                --secret=$(SECRET)

# Remove the Docker image
.PHONY: remove
remove:
        docker rmi $(IMAGE_NAME):dev || echo "Image $(IMAGE_NAME):dev does not exist."