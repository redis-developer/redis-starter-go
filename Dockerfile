# Fetch
FROM golang:1.26.1 AS fetch-stage
WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Build
FROM golang:1.26.1 AS build-stage
WORKDIR /usr/src/app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /usr/local/bin/app cmd/main.go

# Deploy
FROM gcr.io/distroless/static-debian12:nonroot@sha256:a9329520abc449e3b14d5bc3a6ffae065bdde0f02667fa10880c49b35c109fd1 AS deploy-stage
WORKDIR /
COPY --from=build-stage /usr/local/bin/app .
ENV PORT 8080
EXPOSE ${PORT}
ENTRYPOINT [ "/app" ]
