.PHONY: generate format

generate:
	go generate ./...

format:
	find . -name '*.go' -not -path './mock/*' | xargs goimports -w
