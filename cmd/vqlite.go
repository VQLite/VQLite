package main

import (
	"fmt"
	"vqlite/config"
	"vqlite/routers"
)

func main() {
	host := config.GlobalConfig.ServiceConfig.Host
	port := config.GlobalConfig.ServiceConfig.Port
	addr := fmt.Sprintf("%s:%d", host, port)

	r := routes.InitRouter()
	r.Run(addr)
}
