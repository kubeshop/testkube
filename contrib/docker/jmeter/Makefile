# Variables
DOCKER_REPOSITORY = kubeshop
DOCKER_IMAGE_NAME = jmeter
DOCKER_TAG = 5.6.1

# Build the Docker image
.PHONY: build
build:
	@echo "Building Docker image..."
	@docker build -f jmeter5.6.1.ubi8.8.Dockerfile -t $(DOCKER_REPOSITORY)/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG) .

.PHONY: push
push: build
	@echo "Pushing Docker image..."
	@docker push $(DOCKER_REPOSITORY)/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG)

test: build
	@echo "Testing Docker image..."
	@docker run --rm -it $(DOCKER_REPOSITORY)/$(DOCKER_IMAGE_NAME):$(DOCKER_TAG) --version