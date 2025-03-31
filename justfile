piper := 'TTY=0 piper -p piper.cue' + if env("DEBUG", "0") == '1' { ' --log-level=debug' } else { '' }
export KUBECONFIG := home_directory() / ".kube_config/config--algo-staging.yaml"
export PIPER_BUILDER_HOST := ""
crkit := "go tool crkit"

serve:
    {{ crkit }} serve registry -c \
    	--log-format=text \
    	--addr=:5050

serve-proxy:
    {{ crkit }} serve registry -c \
    	--log-format=text \
    	--remote-endpoint=https://${CONTAINER_REGISTRY} \
    	--remote-username=${CONTAINER_REGISTRY_USERNAME} \
    	--remote-password=${CONTAINER_REGISTRY_PASSWORD} \
    	--addr=:5050

dump-k8s:
    {{ crkit }} serve registry --dump-k8s

gen:
    go tool devtool gen --all ./internal/cmd/crkit

dep:
    go get -u ./...

test:
    go test -v -failfast ./...

fmt:
    go tool gofumpt -w -l .

ship:
    $(PIPER) do ship

debug-pull:
    crane pull --format=oci --insecure 0.0.0.0:5050/docker.io/library/nginx:latest .tmp/nginx.tar

debug-pull-proxy:
    crane pull --verbose --format=oci --insecure 0.0.0.0:5050/${CONTAINER_REGISTRY}/gcr.io/distroless/cc-debian12:debug .tmp/ccdebug.tar
    crane pull --format=oci --insecure 0.0.0.0:5050/${CONTAINER_REGISTRY}/ghcr.io/octohelm/crkit:v0.0.0-20241015075301-491947339730 .tmp/crkit.tar

debug-push:
    crane push --insecure .tmp/nginx.tar 0.0.0.0:5050/docker.io/library/nginx:latest
