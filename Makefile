PIPER = TTY=0 piper -p piper.cue
DEBUG = 0
ifeq ($(DEBUG),1)
	PIPER := $(PIPER) --log-level=debug
endif

CRKIT = go run ./internal/cmd/crkit

export KUBECONFIG = ${HOME}/.kube_config/config--algo-staging.yaml
export PIPER_BUILDER_HOST =

serve:
	$(CRKIT) serve registry -c \
		--log-format=text \
		--addr=:5050

dump.k8s:
	$(CRKIT) serve registry --dump-k8s

gen:
	go run ./internal/cmd/tool gen ./internal/cmd/crkit

dep:
	go get -u ./...

test:
	go test -v -failfast ./...

fmt:
	gofumpt -w -l .

ship:
	$(PIPER) do ship

debug.pull:
	crane pull --insecure 0.0.0.0:5050/docker.io/library/nginx:latest .tmp/nginx.tar

debug.push:
	crane push --insecure .tmp/nginx.tar 0.0.0.0:5050/docker.io/library/nginx:latest

