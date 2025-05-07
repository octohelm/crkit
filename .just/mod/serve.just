crkit := "go tool crkit"

serve:
    {{ crkit }} serve registry -c \
    	--log-format=text \
    	--addr=:5070

gc:
    {{ crkit }} gc -c --log-level=debug --dry-run

serve-proxy:
    {{ crkit }} serve registry -c \
    	--log-format=text \
    	--remote-endpoint=https://${CONTAINER_REGISTRY} \
    	--remote-username=${CONTAINER_REGISTRY_USERNAME} \
    	--remote-password=${CONTAINER_REGISTRY_PASSWORD} \
    	--addr=:5070

dump-k8s:
    {{ crkit }} serve registry --dump-k8s
