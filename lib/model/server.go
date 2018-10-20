package model

import (
	"fmt"
	"net"

	"github.com/WangYihang/PrGoxy/lib/util/log"
)

type TCPServer struct {
	Host    string
	Port    int16
	Clients [](*TCPClient)
}

func CreateTCPServer(host string, port int16) *TCPServer {
	return &TCPServer{
		Host:    host,
		Port:    port,
		Clients: make([](*TCPClient), 0),
	}
}

func (o *TCPServer) toString() string {
	return fmt.Sprintf("%s:%d", o.Host, o.Port)
}

func (o *TCPServer) Run() {
	service := fmt.Sprintf("%s:%d", o.Host, o.Port)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
	if err != nil {
		log.Error("Resolve TCP address failed: %s", err)
		return
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Error("Listen failed: %s", err)
		return
	}
	log.Info(fmt.Sprintf("Server running at: %s", o.toString()))
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		client := CreateTCPClient(conn, o)
		log.Info("New client %s Connected", client.toString())
		o.AddTCPClient(client)
		go client.PrGoxy()
	}
}

func (o *TCPServer) DeleteTCPClient(client *TCPClient) {
	client.Close()
	i := 0
	for _, v := range o.Clients {
		if v == client {
			break
		}
		i += 0
	}
	o.Clients = append(o.Clients[:i], o.Clients[i+1:]...)
}

func (o *TCPServer) AddTCPClient(client *TCPClient) {
	o.Clients = append(o.Clients, client)
}
