#
#
# - go targets
###############################
build:
	@echo "Building K8s MCP server..."
	@--mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o k8smcp cmd/k8s-mcp-server/main.go

run: build
	@echo "Running K8s MCP server..."
	@./k8smcp stdio 

clean:
	@echo "Cleaning up..."
	@rm -f k8smcp
	@go clean
	@go clean -modcache

test:
	@echo "Running test..."
	@go test -count=1 -cover -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

#
#
# - brew targets
###############################
brew_k3d_setup:
	brew install k3d
	brew install kubectl
	brew install k9s

brew_k3d_cleanup:
	brew uninstall k3d
	brew uninstall kubectl
	brew uninstall k9s

# To keep using the `docker build` install but with buildkit: https://docs.docker.com/engine/reference/commandline/buildx_install/
brew_docker_buildx_setup:
	brew install docker-buildx
	mkdir -p $(HOME)/.docker/cli-plugins
	ln -sfn $$(brew --prefix docker-buildx) $(HOME)/.docker/cli-plugins/docker-buildx
	docker buildx install

brew_kind_setup:
	brew install kind

#
#
# - k3d targets
###############################
k3d_setup:
	@echo "Installing k3d CLI..."
	curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | TAG=v5.8.3 bash

kubectl_setup:
	@echo "Installing kubectl..."
	curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/arm64/kubectl"
	sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

k3d_setup: k3d_setup kubectl_setup

k3d_cleanup:
	sudo rm $(where kubectl)
	sudo rm $(where k3d)

# https://github.com/k3d-io/k3d/issues/1449#issuecomment-2154672702
K3D_FIX_DNS=0
CLUSTER_NAME?=cluster-one
REGISTRY_PORT?=5432
KUBECONFIG_PATH=$(HOME)/.kube/config
create_k3d_cluster:
	@echo "Creating k3d cluster..."
	K3D_FIX_DNS=0 k3d cluster create $(CLUSTER_NAME) --servers 3 --agents 3 --api-port 0.0.0.0:6550 --registry-create $(CLUSTER_NAME)-registry:0.0.0.0:$(REGISTRY_PORT) --wait --timeout 120s 

delete_k3d_cluster:
	@echo "Deleting k3d cluster..."
	k3d cluster delete $(CLUSTER_NAME)

add_kubeconfig:
	k3d kubeconfig merge $(CLUSTER_NAME) --kubeconfig-switch-context

#
#
# - kind targets
###############################
KIND_CLUSTER_NAME ?= mcp-k8s-test
KIND_CONFIG ?= kind-config.yaml

# Create kind config file if it doesn't exist
$(KIND_CONFIG):
	@echo "Creating kind configuration file..."
	@echo "kind: Cluster" > $(KIND_CONFIG)
	@echo "apiVersion: kind.x-k8s.io/v1alpha4" >> $(KIND_CONFIG)
	@echo "nodes:" >> $(KIND_CONFIG)
	@echo "- role: control-plane" >> $(KIND_CONFIG)
	@echo "- role: worker" >> $(KIND_CONFIG)

# Create a kind cluster
kind_cluster_create: $(KIND_CONFIG)
	@echo "Creating kind cluster $(KIND_CLUSTER_NAME)..."
	kind create cluster --name $(KIND_CLUSTER_NAME) --config $(KIND_CONFIG)
	@echo "Cluster created successfully!"
	@echo "Configuring kubectl context..."
	kind export kubeconfig --name $(KIND_CLUSTER_NAME)
	@echo "Ready to use the cluster!"

# Delete the kind cluster
kind_cluster_delete:
	@echo "Deleting kind cluster $(KIND_CLUSTER_NAME)..."
	kind delete cluster --name $(KIND_CLUSTER_NAME)
	@echo "Cluster deleted successfully!"

# Get cluster status
kind_cluster_status:
	@if kind get clusters | grep -q $(KIND_CLUSTER_NAME); then \
		echo "Cluster $(KIND_CLUSTER_NAME) is running"; \
		kubectl --context kind-$(KIND_CLUSTER_NAME) get nodes; \
	else \
		echo "Cluster $(KIND_CLUSTER_NAME) is not running"; \
	fi

# Deploy test resources to the kind cluster
deploy_test_resources:
	@echo "Deploying test resources to kind cluster..."
	kubectl --context kind-$(KIND_CLUSTER_NAME) apply -f test/resources/

# Run MCP server against the kind cluster
run_with_kind: build
	@echo "Starting MCP server connected to kind cluster..."
	KUBECONFIG=$(KUBECONFIG_PATH) ./k8smcp stdio

# Complete test cycle with kind cluster
kind_test_cycle: kind_cluster_create deploy_test_resources test kind_cluster_delete

#
#
# - docker targets
###############################
IMAGE_TAG?=$(shell date +%Y%m%d%H%M%S)
build_container:
	@dir_name="$$(basename "$$PWD")"; \
	DOCKER_BUILDKIT=1 docker build . --tag $$dir_name:$(IMAGE_TAG)

run_container:
	@dir_name="$$(basename "$$PWD")"; \
	docker build . --tag $$dir_name:$(IMAGE_TAG); \
	docker run -it -p 8080:8080 $$dir_name:$(IMAGE_TAG)

# Build image and push to dedicated k3d-managed registry and
# Update deployment image tag
build_push_image:
	@dir_name="$$(basename "$$PWD")"; \
	image_tag="$(IMAGE_TAG)"; \
	image_repo="$(CLUSTER_NAME)-registry.localhost:$(REGISTRY_PORT)/$$dir_name:$$image_tag"; \
	echo "building image $$dir_name:$$image_tag..."; \
	docker build . --tag $$dir_name:$$image_tag; \
	docker tag $$dir_name:$$image_tag $$image_repo; \
	docker push $$image_repo; \
	echo "Updating image tag in k8s/deployment.yaml to $$image_tag..."; \
	sed -i.bak -E "s|(image: .+):[^:]+$$|\1:$$image_tag|" k8s/deployment.yaml

# Load Docker image into kind cluster
kind_load_image: build_container
	@dir_name="$$(basename "$$PWD")"; \
	kind load docker-image $$dir_name:$(IMAGE_TAG) --name $(KIND_CLUSTER_NAME)

#
#
# - help
###############################
.PHONY: help
help:
	@echo "Kubernetes MCP Server Development Makefile"
	@echo ""
	@echo "Go Targets:"
	@echo "  build                    - Build the K8s MCP server"
	@echo "  run                      - Run the K8s MCP server (stdio)"
	@echo "  clean                    - Clean up build artifacts"
	@echo "  test                     - Run tests and generate coverage"
	@echo ""
	@echo "Brew Targets:"
	@echo "  brew_k3d_setup           - Install k3d, kubectl, and k9s via Homebrew"
	@echo "  brew_k3d_cleanup         - Uninstall k3d, kubectl, and k9s"
	@echo "  brew_docker_buildx_setup - Install Docker BuildKit"
	@echo "  brew_kind_setup          - Install kind via Homebrew"
	@echo ""
	@echo "K3D Targets:"
	@echo "  k3d_setup                - Install k3d CLI"
	@echo "  kubectl_setup            - Install kubectl"
	@echo "  create_k3d_cluster       - Create a k3d cluster"
	@echo "  delete_k3d_cluster       - Delete the k3d cluster"
	@echo "  add_kubeconfig           - Update kubeconfig for k3d cluster"
	@echo ""
	@echo "Kind Targets:"
	@echo "  kind_cluster_create      - Create a kind cluster for testing"
	@echo "  kind_cluster_delete      - Delete the kind cluster"
	@echo "  kind_cluster_status      - Check kind cluster status"
	@echo "  deploy_test_resources    - Deploy test resources to kind cluster"
	@echo "  run_with_kind            - Run MCP server against kind cluster"
	@echo "  kind_test_cycle          - Full test cycle with kind (create, test, delete)"
	@echo "  kind_load_image          - Load Docker image into kind cluster"
	@echo ""
	@echo "Docker Targets:"
	@echo "  build_container          - Build Docker container"
	@echo "  run_container            - Build and run Docker container"
	@echo "  build_push_image         - Build and push image to k3d registry"
	@echo ""
	@echo "Configuration Variables:"
	@echo "  KIND_CLUSTER_NAME        - Name of the kind cluster (default: mcp-k8s-test)"
	@echo "  KIND_CONFIG              - Path to kind config file (default: kind-config.yaml)"
	@echo "  CLUSTER_NAME             - Name of the k3d cluster (default: cluster-one)"
	@echo "  REGISTRY_PORT            - Port for the k3d registry (default: 5432)"
	@echo "  KUBECONFIG_PATH          - Path to kubeconfig file (default: ~/.kube/config)"
	@echo "  IMAGE_TAG                - Tag for Docker images (default: timestamp)"