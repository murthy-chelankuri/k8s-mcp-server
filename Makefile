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
	@go test -count=1 -coverprofile=coverage.out ./...
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
# - docker targets
###############################
install_buildx:


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