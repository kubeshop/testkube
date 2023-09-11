# Variables
DOCKER_REPOSITORY = kubeshop
DOCKER_IMAGE_NAME = jmeter
DOCKER_TAG = 5.5

# Build the Docker image
.PHONY: build
build:
	@echo "Building Docker image..."
	@docker buildx build --platform linux/amd64,linux/arm64 -f jmeter5.5.ubi8.8.Dockerfile -t $(DOCKER_REPOSITORY)/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG) .

.PHONY: push
push: build
	@echo "Pushing Docker image..."
	@docker buildx build --push --platform linux/amd64,linux/arm64 -f jmeter5.5.ubi8.8.Dockerfile -t $(DOCKER_REPOSITORY)/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG) .

test: build
	@echo "Testing Docker image..."
	@docker run --rm -it $(DOCKER_REPOSITORY)/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG) --version