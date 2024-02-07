generate:
	go generate ./...

lint: generate
	gofumpt -l -w .
	goimports -local github.com/mt-inside/polybar-timezones -w .
	go vet ./...
	staticcheck ./...
	golangci-lint run ./...

run:
	go run main.go

install:
	go install .
