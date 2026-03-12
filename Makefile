MAKEFLAGS += --no-print-directory

## ---------------------------------------------------------------------------
## | The purpose of this Makefile is to provide all the functionality needed |
## | to install, develop, build, and run this app.                           |
## ---------------------------------------------------------------------------

help:              ## Show this help
	@sed -ne '/@sed/!s/## //p' $(MAKEFILE_LIST)

install:           ## Install all dependencies
	@go mod tidy

build:             ## Build a binary for the app
	@$(MAKE) install
	@go build -o ./bin/app cmd/main.go

dev:               ## Run the dev server
	@$(MAKE) install
	@go run cmd/main.go

serve:             ## Build and run the production binary
	@$(MAKE) build
	@./bin/app

test:              ## Run tests
	@$(MAKE) install
	@go test -p 1 ./... -v

test-short:        ## Run tests without Docker-backed integration coverage
	@$(MAKE) install
	@go test -short -p 1 ./... -v

docker:            ## Spin down docker containers and then rebuild and run them
	@docker compose down
	@docker compose up -d --build

format:            ## Format code
	@gofmt -s -w .

list-updates:      ## List updates to go dependencies
	@go list -m -u -json all

update:            ## Update all go dependencies and install
	@go get github.com/labstack/echo/v4@v4.15.1 \
		github.com/redis/go-redis/v9@v9.18.0 \
		github.com/stretchr/testify@v1.11.1 \
		github.com/testcontainers/testcontainers-go@v0.41.0
	@go get -u ./...
	@$(MAKE) install

clean:             ## Remove build files
	@rm -rf bin
