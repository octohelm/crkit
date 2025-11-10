devtool := 'go tool devtool'

gen:
    {{ devtool }} gen --all ./internal/cmd/crkit

update:
    go get -u ./...

dep:
    go mod tidy

test:
    go test -count=1 -failfast ./...

test-race:
    CGO_ENABLED=1 go test -count=1 -failfast -race ./...

fmt:
    {{ devtool }} fmt -w -l .
