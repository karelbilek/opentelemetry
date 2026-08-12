generate:
	go tool -modfile=tools.mod buf generate
	go mod tidy
	go build ./...

.PHONY: generate
