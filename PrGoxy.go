package main

import (
	"time"

	"github.com/WangYihang/PrGoxy/lib/config"
	"github.com/WangYihang/PrGoxy/lib/model"
)

func main() {
	// Sync config.json
	go func() {
		for {
			config.Cfg.Reload()
			time.Sleep(time.Second * 3)
		}
	}()
	// Start server
	server := model.CreateTCPServer(
		config.Cfg.Proxy.LHost,
		config.Cfg.Proxy.LPort,
	)
	server.Run()
}
