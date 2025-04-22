set allow-duplicate-variables := true

import? '.just/local.just'
import '.just/default.just'
import '.just/mod/ci.just'
import '.just/mod/go.just'
import '.just/mod/serve.just'

debug-pull:
    crane pull --format=oci --insecure ${CONTAINER_REGISTRY}/ghcr.io/octohelm/crkit:v0.0.0-20241015075301-491947339730 .tmp/crkit.tar

debug-pull-proxy:
    crane pull --verbose --format=oci --insecure 0.0.0.0:5070/${CONTAINER_REGISTRY}/gcr.io/distroless/cc-debian12:debug .tmp/ccdebug.tar
    crane pull --format=oci --insecure 0.0.0.0:5070/${CONTAINER_REGISTRY}/ghcr.io/octohelm/crkit:v0.0.0-20241015075301-491947339730 .tmp/crkit.tar

debug-push:
    crane push --insecure .tmp/crkit.tar 0.0.0.0:5070/octohelm/crkit:latest
