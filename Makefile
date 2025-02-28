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

dev:               ## Run a dev server and watch files to rebuild
	@$(MAKE) install
	@go run cmd/main.go

serve:             ## Build and run the production binary
	@$(MAKE) build
	@./bin/app

test:              ## Run tests
	@$(MAKE) install
	@go test ./... -v

docker:            ## Spin down docker containers and then rebuild and run them
	@docker compose down
	@docker compose up -d --build

format:            ## Format code
	@gofmt -s -w .

list-updates:      ## List updates to go dependencies
	@go list -m -u all

update:            ## Update all go dependencies and install
	@go get -u ./...
	@$(MAKE) install

clean:             ## Remove build files
	@rm -rf bin
