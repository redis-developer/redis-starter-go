build: install
	@go build -o ./bin/app cmd/main.go

install:
	@go mod tidy

dev: install
	@go run cmd/main.go

run: build
	./bin/app

test: install
	@go test pkg/components/todos/store_test.go pkg/components/todos/store.go -v

docker:
	@docker compose up -d

clean:
	@rm -rf bin
