package main

import (
	"github.com/redis-developer/redis-starter-go/cmd/config"
	"github.com/redis-developer/redis-starter-go/pkg"
)

func main() {
	server.New(config.New()).Run()
}
