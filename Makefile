WAGON = wagon -p wagon.cue
DEBUG = 0
ifeq ($(DEBUG),1)
	WAGON := $(WAGON) --log-level=debug
endif

export BUILDKIT_HOST =

CRKIT = go run ./internal/cmd/crkit

dump.k8s:
	$(CRKIT) serve registry --dump-k8s

serve.registry:
	$(CRKIT) serve registry \
		--storage-root=.tmp/container-registry \
		--remote-registry-endpoint=https://${CONTAINER_REGISTRY} \
		--remote-registry-username=${CONTAINER_REGISTRY_USERNAME} \
		--remote-registry-password=${CONTAINER_REGISTRY_PASSWORD} \
		--addr=:5050

gen:
	go run ./internal/cmd/tool gen ./internal/cmd/crkit

test:
	go test -v -failfast ./...

ship:
	$(WAGON) do go ship pushx

debug.pull:
	crane pull --insecure 0.0.0.0:5050/docker.io/library/nginx:latest .tmp/nginx.tar

debug.push:
	crane push --insecure .tmp/nginx.tar 0.0.0.0:5050/docker.io/library/nginx:latest
