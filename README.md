This is a [Redis](https://redis.io/) starter template for Go:

- [Redis Cloud](https://redis.io/try-free/)
- [Echo](https://echo.labstack.com/)

## Requirements

- [make](https://www.make.com/en)
- [go>=1.26.1](https://go.dev/)
- [docker](https://www.docker.com/)
   - Optional

## Getting started

Copy and edit the `.env` file:

```bash
cp .env.example .env
```

Your `.env` file should contain the connection string you copied from Redis Cloud.
The additional settings are optional:

- `APP_ENV` defaults to `development` and also accepts `test` or `production`
- `LOG_LEVEL` defaults to `info`
- `LOG_STREAM_KEY` defaults to `logs`

If `APP_ENV` is unset, `NODE_ENV` is accepted as a fallback alias.

Your `.env.docker` file will look similar to `.env`, but should use the appropriate docker internal URLs. Here is
an example:

```bash
HOST_PORT=8081
REDIS_URL="redis://redis:6379"
```

Next, spin up docker containers:

```bash
make docker
```

You should have a server running on `http://localhost:<host-port>` where `HOST_PORT` comes from `.env.docker` (default is 8081). The app still listens on the internal `PORT` value (default 8080). You can test the following routes:

1. `GET /api/todos` - Gets all todos
2. `GET /api/todos/:id` - Gets a todo by ID
3. `GET /api/todos/search?[name=<name>]&[status=<status>]` - Search for todos by name and/or status
4. `POST /api/todos` - Create a todo with `{ "name": "Sample todo" }`
5. `PATCH /api/todos/:id` - Update todo by ID with `{ "status": "todo|in progress|complete" }`
6. `DELETE /api/todos/:id` - Delete a todo by ID

## Running tests

There are unit and Docker-backed integration tests that can be run with the following command:

```bash
make test
```

These tests setup and teardown on their own with Testcontainers. If you want to skip the Docker-backed coverage, run:

```bash
make test-short
```

## Running locally outside docker

To run the development server outside of docker:

```bash
make dev
```

## Other Scripts

Run `make` to see the list of available commands. Application and request logs are written to stdout and also appended to the Redis stream named by `LOG_STREAM_KEY`.

## Connecting to Redis Cloud

If you don't yet have a database setup in Redis Cloud [get started here for free](https://redis.io/try-free/).

To connect to a Redis Cloud database, log into the console and find the following:

1. The `public endpoint` (looks like `redis-#####.c###.us-east-1-#.ec2.redns.redis-cloud.com:#####`)
1. Your `username` (`default` is the default username, otherwise find the one you setup)
1. Your `password` (either setup through Data Access Control, or available in the `Security` section of the database
   page.

Combine the above values into a connection string and put it in your `.env` and `.env.docker` accordingly. It should
look something like the following:

```bash
REDIS_URL="redis://default:<password>@redis-#####.c###.us-west-2-#.ec2.redns.redis-cloud.com:#####"
```

Run the [tests](#running-tests) to verify that you are connected properly.

## Learn more

To learn more about Redis, take a look at the following resources:

- [Redis Documentation](https://redis.io/docs/latest/) - learn about Redis products, features, and commands.
- [Learn Redis](https://redis.io/learn/) - read tutorials, quick starts, and how-to guides for Redis.
- [Redis Demo Center](https://redis.io/demo-center/) - watch short, technical videos about Redis products and features.
