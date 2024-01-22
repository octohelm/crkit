WAGON = wagon -p wagon.cue
DEBUG = 0
ifeq ($(DEBUG),1)
	WAGON := $(WAGON) --log-level=debug
endif

export BUILDKIT_HOST =

dump.k8s:
	go run ./internal/cmd/crkit serve registry --dump-k8s

serve.registry:
	go run ./internal/cmd/crkit serve registry

gen:
	go run ./internal/cmd/tool gen ./internal/cmd/crkit

test:
	go test -v -failfast ./...

ship:
	$(WAGON) do go ship pushx
