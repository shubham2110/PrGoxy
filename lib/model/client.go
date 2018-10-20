package model

import (
	"bytes"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/WangYihang/PrGoxy/lib/util/log"
)

type TCPClient struct {
	Conn      net.Conn
	ReadLock  *sync.Mutex
	WriteLock *sync.Mutex
	Server    *TCPServer
	Request   string
}

func CreateTCPClient(conn net.Conn, server *TCPServer) *TCPClient {
	return &TCPClient{
		Conn:      conn,
		ReadLock:  new(sync.Mutex),
		WriteLock: new(sync.Mutex),
		Server:    server,
		Request:   "",
	}
}
func (o *TCPClient) ToString() string {
	return o.Conn.RemoteAddr().String()
}

func (o *TCPClient) Close() {
	log.Info("Closeing client: %s", o.ToString())
	o.Conn.Close()
}
func (o *TCPClient) ReadUntil(token string) string {
	inputBuffer := make([]byte, 1)
	var outputBuffer bytes.Buffer
	for {
		o.ReadLock.Lock()
		n, err := o.Conn.Read(inputBuffer)
		o.ReadLock.Unlock()
		if err != nil {
			log.Error("Read from client failed")
			o.Server.DeleteTCPClient(o)
			return outputBuffer.String()
		}
		outputBuffer.Write(inputBuffer[:n])
		// If found token, then finish reading
		if strings.HasSuffix(outputBuffer.String(), token) {
			break
		}
	}
	log.Info("%d bytes read from client", len(outputBuffer.String()))
	return outputBuffer.String()
}

func (o *TCPClient) ReadSize(size int) string {
	o.Conn.SetReadDeadline(time.Now().Add(time.Second * 3))
	readSize := 0
	inputBuffer := make([]byte, 1)
	var outputBuffer bytes.Buffer
	for {
		o.ReadLock.Lock()
		n, err := o.Conn.Read(inputBuffer)
		o.ReadLock.Unlock()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Error("Read response timeout from client")
			} else {
				log.Error("Read from client failed")
				o.Server.DeleteTCPClient(o)
			}
			break
		}
		// If read size equals zero, then finish reading
		outputBuffer.Write(inputBuffer[:n])
		readSize += n
		if readSize >= size {
			break
		}
	}
	log.Info("(%d/%d) bytes read from client", len(outputBuffer.String()), size)
	return outputBuffer.String()
}

func (o *TCPClient) Read(timeout time.Duration) (string, bool) {
	// Set read time out
	o.Conn.SetReadDeadline(time.Now().Add(timeout))

	inputBuffer := make([]byte, 1024)
	var outputBuffer bytes.Buffer
	var isTimeout bool
	for {
		o.ReadLock.Lock()
		n, err := o.Conn.Read(inputBuffer)
		o.ReadLock.Unlock()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				isTimeout = true
			} else {
				log.Error("Read from client failed")
				o.Server.DeleteTCPClient(o)
				isTimeout = false
			}
			break
		}
		outputBuffer.Write(inputBuffer[:n])
	}
	// Reset read time out
	o.Conn.SetReadDeadline(time.Time{})

	return outputBuffer.String(), isTimeout
}

func (o *TCPClient) Write(data []byte) int {
	o.WriteLock.Lock()
	n, err := o.Conn.Write(data)
	o.WriteLock.Unlock()
	if err != nil {
		log.Error("Write to client failed")
		o.Server.DeleteTCPClient(o)
	}
	log.Info("%d bytes sent to client", n)
	return n
}

func (o *TCPClient) PrGoxy() {

}
