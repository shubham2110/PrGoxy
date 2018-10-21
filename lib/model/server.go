package model

import (
	"container/list"
	"fmt"
	"net"

	"github.com/WangYihang/PrGoxy/lib/util/log"
)

type TCPServer struct {
	Host    string
	Port    int16
	Clients *list.List
}

func CreateTCPServer(host string, port int16) *TCPServer {
	return &TCPServer{
		Host:    host,
		Port:    port,
		Clients: list.New(),
	}
}

func (o *TCPServer) ToString() string {
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
	log.Info(fmt.Sprintf("Server running at: %s", o.ToString()))
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		client := CreateTCPClient(conn, o)
		log.Debug("New client %s Connected", client.ToString())
		o.AddTCPClient(client)
		go client.PrGoxy()
	}
}

func Contains(l *list.List, value *TCPClient) *list.Element {
	for e := l.Front(); e != nil; e = e.Next() {
		if e.Value == value {
			return e
		}
	}
	return nil
}

func (o *TCPServer) DeleteTCPClient(client *TCPClient) {
	defer client.Close()
	if e := Contains(o.Clients, client); e != nil {
		o.Clients.Remove(e)
	}
}

func (o *TCPServer) AddTCPClient(client *TCPClient) {
	o.Clients.PushBack(client)
}
