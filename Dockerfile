# Fetch
FROM golang:latest AS fetch-stage
WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Build
FROM golang:latest AS build-stage
WORKDIR /usr/src/app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /usr/local/bin/app cmd/main.go

# Deploy
FROM gcr.io/distroless/static-debian12 AS deploy-stage
WORKDIR /
COPY --chown=nonroot --from=build-stage /usr/local/bin/app .
ENV PORT 8080
EXPOSE ${PORT}
USER nonroot:nonroot
ENTRYPOINT [ "/app" ]

