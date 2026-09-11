.DEFAULT_GOAL := test

.PHONY: build fmt generate proto test tidy verify

build:
	go build ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './internal/neoshowcase/gen/*')

proto:
	cd proto && go run github.com/bufbuild/buf/cmd/buf@v1.72.0 generate

generate: proto
	go generate ./...

test:
	go test ./...

tidy:
	go mod tidy

verify: fmt tidy test
	git diff --exit-code
