# Runs the frontend tooling inside Docker. Run `make help` for the target list.

IMAGE_TAG ?= statsviz-ui
NODE_IMAGE ?= node:24-alpine
NPM_CACHE_VOLUME ?= statsviz-npm-cache
# Complete toolchain image, including Node, npm and zip. Release requires a digest.
RELEASE_IMAGE ?=
PORT ?= 5173

UID := $(shell id -u)
GID := $(shell id -g)

# Common `docker run` flags. `docker run` argument order is:
#   docker run [OPTIONS] IMAGE [COMMAND] [ARG...]
# so any per-target flags (like `-p`) must come BEFORE $(IMAGE_TAG) in the
# recipe. Do not append the image tag to these variables.
DOCKER_FLAGS := --rm \
	-v "$(CURDIR)":/app \
	-v $(NPM_CACHE_VOLUME):/npm-cache \
	-w /app \
	-e HOST_UID=$(UID) \
	-e HOST_GID=$(GID)

.PHONY: help image dev build zip release install run shell clean distclean

help:
	@awk 'BEGIN{FS=":.*##"; printf "Targets:\n"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

image: ## Build the Docker image (Docker reuses unchanged layers)
	docker build --build-arg NODE_IMAGE=$(NODE_IMAGE) -t $(IMAGE_TAG) .

dev: image ## Start the Vite dev server on http://localhost:$(PORT)
	docker run -it $(DOCKER_FLAGS) -p $(PORT):5173 $(IMAGE_TAG) \
		sh -c 'npm install && npm run dev -- --host 0.0.0.0 --port 5173'

install: image ## Run `npm install` in the container
	docker run -i $(DOCKER_FLAGS) $(IMAGE_TAG) npm install

build: image ## Install the lockfile and build
	docker run -i $(DOCKER_FLAGS) $(IMAGE_TAG) sh -c 'npm ci --no-audit --no-fund && npm run build'

zip: build ## Build and regenerate dist.zip
	docker run -i $(DOCKER_FLAGS) $(IMAGE_TAG) sh scripts/zip.sh

release: ## Offline release using a digest-pinned complete toolchain image
	@case "$(RELEASE_IMAGE)" in sha256:*|*@sha256:*) ;; *) echo 'Set RELEASE_IMAGE to an immutable image digest' >&2; exit 2;; esac
	docker image inspect "$(RELEASE_IMAGE)" >/dev/null
	docker run -i --pull=never --network=none $(DOCKER_FLAGS) -e STATSVIZ_RELEASE_IMAGE="$(RELEASE_IMAGE)" "$(RELEASE_IMAGE)" sh scripts/release.sh

run: image ## Run an arbitrary command in the container: make run CMD="npm outdated"
	@if [ -z "$(CMD)" ]; then \
		echo "usage: make run CMD=\"<command>\"" >&2; exit 2; \
	fi
	docker run -it $(DOCKER_FLAGS) $(IMAGE_TAG) sh -c '$(CMD)'

shell: image ## Open an interactive shell in the container
	docker run -it $(DOCKER_FLAGS) $(IMAGE_TAG) sh

clean: ## Remove dist/, dist.zip, node_modules/
	rm -rf dist dist.zip node_modules

distclean: clean ## Also remove the Docker image and npm cache volume
	-docker image rm $(IMAGE_TAG) 2>/dev/null || true
	-docker volume rm $(NPM_CACHE_VOLUME) 2>/dev/null || true
	rm -f .docker-image-stamp
