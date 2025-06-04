# Docker-related targets

# Check if required variables are set
check-docker-vars:
	@if [ -z "$(ORG)" ]; then \
		echo "Error: ORG variable is required"; \
		echo "Usage: make build-images ORG=iddaa-backend TAG=latest"; \
		exit 1; \
	fi
	@if [ -z "$(TAG)" ]; then \
		echo "Error: TAG variable is required"; \
		echo "Usage: make build-images ORG=iddaa-backend TAG=latest"; \
		exit 1; \
	fi

build-api-image: ## Build API Docker image
	@echo "Building API Docker image..."
	docker build -f docker/Dockerfile.api -t iddaa-api .

build-cron-image: ## Build cron Docker image
	@echo "Building cron Docker image..."
	docker build -f docker/Dockerfile.cron -t iddaa-cron .

build-images: check-docker-vars ## Build all Docker images with tags (ORG=org TAG=tag)
	@echo "Building Docker images $(ORG)/api:$(TAG) and $(ORG)/cron:$(TAG)..."
	docker build -f docker/Dockerfile.api -t $(ORG)/api:$(TAG) .
	docker build -f docker/Dockerfile.cron -t $(ORG)/cron:$(TAG) .

push-images: check-docker-vars ## Push all Docker images (ORG=org TAG=tag)
	@echo "Pushing Docker images $(ORG)/api:$(TAG) and $(ORG)/cron:$(TAG)..."
	docker push $(ORG)/api:$(TAG)
	docker push $(ORG)/cron:$(TAG)

build-and-push: build-images push-images ## Build and push all images (ORG=org TAG=tag)

# Individual image operations
build-api-tagged: check-docker-vars ## Build tagged API image (ORG=org TAG=tag)
	@echo "Building API Docker image $(ORG)/api:$(TAG)..."
	docker build -f docker/Dockerfile.api -t $(ORG)/api:$(TAG) .

build-cron-tagged: check-docker-vars ## Build tagged cron image (ORG=org TAG=tag)
	@echo "Building cron Docker image $(ORG)/cron:$(TAG)..."
	docker build -f docker/Dockerfile.cron -t $(ORG)/cron:$(TAG) .

push-api: check-docker-vars ## Push API image (ORG=org TAG=tag)
	@echo "Pushing API Docker image $(ORG)/api:$(TAG)..."
	docker push $(ORG)/api:$(TAG)

push-cron: check-docker-vars ## Push cron image (ORG=org TAG=tag)
	@echo "Pushing cron Docker image $(ORG)/cron:$(TAG)..."
	docker push $(ORG)/cron:$(TAG)

.PHONY: check-docker-vars build-api-image build-cron-image build-images push-images build-and-push build-api-tagged build-cron-tagged push-api push-cron