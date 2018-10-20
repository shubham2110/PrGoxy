package main

import "github.com/WangYihang/PrGoxy/lib/model"

func main() {
	// Load config
	host := "127.0.0.1"
	port := 8080
	// Start server
	server := model.CreateTCPServer(host, int16(port))
	server.Run()
}
