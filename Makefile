PIPER = piper -p piper.cue
DEBUG = 0
ifeq ($(DEBUG),1)
	PIPER := $(PIPER) --log-level=debug
endif

CRKIT = go run ./internal/cmd/crkit

export KUBECONFIG = ${HOME}/.kube_config/config--algo-staging.yaml

serve.registry:
	$(CRKIT) serve registry \
		--kubeconfig=${KUBECONFIG} \
		--public-ip=10.244.0.175 \
		--storage-root=.tmp/container-registry \
		--addr=:5050

serve.operator:
	$(CRKIT) serve operator -c \
		--containerd-host-config-path=target/containerd/certs.d/ \
		--kubeconfig=${KUBECONFIG} \
		--watch-namespace=kube-system

dump.k8s:
	$(CRKIT) serve registry --dump-k8s
	$(CRKIT) serve operator --dump-k8s

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

