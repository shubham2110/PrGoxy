package model

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WangYihang/PrGoxy/lib/util/log"
)

type TCPClient struct {
	Conn       net.Conn
	ReadLock   *sync.Mutex
	WriteLock  *sync.Mutex
	Server     *TCPServer
	Request    *http.Request
	RawRequest string
}

func CreateTCPClient(conn net.Conn, server *TCPServer) *TCPClient {
	return &TCPClient{
		Conn:       conn,
		ReadLock:   new(sync.Mutex),
		WriteLock:  new(sync.Mutex),
		Server:     server,
		Request:    &http.Request{},
		RawRequest: "",
	}
}
func (o *TCPClient) ToString() string {
	return o.Conn.RemoteAddr().String()
}

func (o *TCPClient) ResponseAndAbort(response string) {
	o.Write([]byte(response))
	o.Server.DeleteTCPClient(o)
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
	// Client handler
	o.ClientFilterHandler()

	// Read whole request
	o.RawRequest = o.ReadUntil("\r\n\r\n")
	// Parse request
	request, err := http.ReadRequest(bufio.NewReader(strings.NewReader(o.RawRequest)))
	if err != nil {
		log.Error("Parse request failed")
		o.ResponseAndAbort("Invalid request")
	}
	o.Request = request

	// Website handler
	o.SiteFilterHandler()
	// Redirect handler
	o.RedirectHandler()
	// Cache handler
	o.CacheHandler()
	// Proxy handler
	o.ProxyHandler()
}

func (o *TCPClient) ClientFilterHandler() {
	// for _, v := range(config.hosts) {
	// 	// check if 127.0.0.1:22537 starts with 127.0.0.1
	// 	if strings.HasPrefix(o.Conn.RemoteAddr().String(), v) {
	// 		// blocked
	// 		o.ResponseAndAbort("Not allowed")
	// 	}
	// }
}

func (o *TCPClient) SiteFilterHandler() {

}

func (o *TCPClient) RedirectHandler() {

}

func (o *TCPClient) CacheHandler() {

}

func (o *TCPClient) ProxyHandler() {
	var err error
	/*
		type URL struct {
			Scheme     string
			Opaque     string    // encoded opaque data
			User       *Userinfo // username and password information
			Host       string    // host or host:port
			Path       string    // path (relative paths may omit leading slash)
			RawPath    string    // encoded path hint (see EscapedPath method)
			ForceQuery bool      // append a query ('?') even if RawQuery is empty
			RawQuery   string    // encoded query values, without '?'
			Fragment   string    // fragment for references, without '#'
		}
	*/
	/*
		log.Info("%s", o.RawRequest)
		log.Info("%s", o.Request.URL)
		log.Info("%s", o.Request.URL.Scheme)
		log.Info("%s", o.Request.URL.Opaque)
		log.Info("%s", o.Request.URL.User)
		log.Info("%s", o.Request.URL.Host)
		log.Info("%s", o.Request.URL.Path)
		log.Info("%s", o.Request.URL.RawPath)
		log.Info("%s", o.Request.URL.ForceQuery)
		log.Info("%s", o.Request.URL.RawQuery)
		log.Info("%s", o.Request.URL.Fragment)
	*/
	// Check scheme
	if o.Request.URL.Scheme != "http" {
		o.ResponseAndAbort("Invalid scheme")
	}
	// Purify Host && Port
	dst := strings.Split(o.Request.URL.Host, ":")
	host := dst[0]
	port := 80
	if len(dst) > 1 {
		port, err = strconv.Atoi(dst[1])
		if err != nil {
			o.ResponseAndAbort("Invalid port")
		}
	}
	log.Info("Destination: %s:%d", host, port)
	// Add X-Forwarded-For Header
	o.Request.Header["X-Forwarded-For"] = []string{"127.0.0.1"}
	// Open port of dst host
	// Transfer data
	o.Request.Write(os.Stdout)
}
