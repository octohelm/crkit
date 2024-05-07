PIPER = piper -p piper.cue
DEBUG = 0
ifeq ($(DEBUG),1)
	PIPER := $(PIPER) --log-level=debug
endif

CRKIT = go run ./internal/cmd/crkit

serve.registry:
	$(CRKIT) serve registry \
		--storage-root=.tmp/container-registry \
		--addr=:5050

dump.k8s:
	$(CRKIT) serve registry --dump-k8s

gen:
	go run ./internal/cmd/tool gen ./internal/cmd/crkit

dep:
	go get -u ./...

test:
	go test -v -failfast ./...

ship:
	$(PIPER) do ship

debug.pull:
	crane pull --insecure 0.0.0.0:5050/docker.io/library/nginx:latest .tmp/nginx.tar

debug.push:
	crane push --insecure .tmp/nginx.tar 0.0.0.0:5050/docker.io/library/nginx:latest

