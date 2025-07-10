-include .env

SHA=`git show --quiet --format=format:%H`

build:
	CGO_ENABLED=0 go build -o app cmd/main.go

build_azure:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X github.com/lildude/starling-sweep/internal/ping.Version=$(SHA)" -o app cmd/main.go

lint:
	golangci-lint run

test:
	ENV=test go test -p 8 ./...

coverage:
	ENV=test go test ./... -coverprofile=coverage.out
	go tool cover -func coverage.out

start: build
	ENV=dev func start

last-uid:
	echo GET starling_webhookevent_uid | redis-cli -u ${REDIS_URL}

get-auth-token:
	echo GET strava_auth_token | redis-cli -u ${REDIS_URL} --no-auth-warning | jq

# Really not sure which of these get things working, but it should produce something like:
# {
#   "clientId": "...",
#   "clientSecret": "...",
#   "subscriptionId": "...",
#   "tenantId": "...",
#   "activeDirectoryEndpointUrl": "https://login.microsoftonline.com",
#   "resourceManagerEndpointUrl": "https://management.azure.com/",
#   "activeDirectoryGraphResourceId": "https://graph.windows.net/",
#   "sqlManagementEndpointUrl": "https://management.core.windows.net:8443/",
#   "galleryEndpointUrl": "https://gallery.azure.com/",
#   "managementEndpointUrl": "https://management.core.windows.net/"
# }
# Set this in AZURE_RBAC_CREDENTIALS in GitHub Actions secrets
new-azure-creds:
	az ad sp create-for-rbac --name "Starling Sweeper" --role contributor \
    --scopes /subscriptions/${AZURE_SUBSCRIPTION_ID}/resourceGroups/starling-sweeper/providers/Microsoft.Web/sites/starling-sweeper \
    --json-auth
