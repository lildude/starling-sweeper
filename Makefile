-include .env

SHA=`git show --quiet --format=format:%H`

build:
	go build -o app cmd/main.go

build_azure:
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X github.com/lildude/starling-sweep/internal/ping.Version=$(SHA)" -o app cmd/main.go

lint:
	golangci-lint run --timeout=20m

test:
	go test -v ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func coverage.out

start: build
	func start

last-uid:
	echo GET starling_webhookevent_uid | redis-cli -u ${REDIS_URL}