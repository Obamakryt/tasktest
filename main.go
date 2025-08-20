package main

import (
	"fmt"
	cfg "gomodlag/internal/config"
	"gomodlag/internal/server"
)

func main() {
	err := cfg.InitEnv()
	config, err := cfg.ItitConfig()
	fmt.Println(config, err)

	server.Start(*config)
}
