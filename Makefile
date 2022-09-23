build:
	go build -o app cmd/main.go

build_azure:
	GOOS=linux GOARCH=amd64 go build -o app cmd/main.go

lint:
	golangci-lint run --timeout=20m

test:
	go test -v ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func coverage.out

start:
	func start
