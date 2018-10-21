package model

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WangYihang/PrGoxy/lib/config"
	"github.com/WangYihang/PrGoxy/lib/util/log"
)

type HTTPRequest struct {
	Method      string
	RequestURI  *url.URL
	HTTPVersion string
	Headers     map[string]string
	Body        string
}

type HTTPResponse struct {
	HTTPVersion  string
	StatusCode   int
	ReasonPhrase string
	Headers      map[string]string
	Body         string
}

type TCPClient struct {
	Conn      net.Conn
	ReadLock  *sync.Mutex
	WriteLock *sync.Mutex
	Server    *TCPServer
	Request   *HTTPRequest
}

func CreateTCPClient(conn net.Conn, server *TCPServer) *TCPClient {
	return &TCPClient{
		Conn:      conn,
		ReadLock:  new(sync.Mutex),
		WriteLock: new(sync.Mutex),
		Server:    server,
		Request: &HTTPRequest{
			Headers: make(map[string]string),
		},
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
	log.Debug("Closeing client: %s", o.ToString())
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
	log.Debug("%d bytes read from client", len(outputBuffer.String()))
	return outputBuffer.String()
}

func (o *TCPClient) ReadUntilClean(token string) string {
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
	log.Debug("%d bytes read from client", len(outputBuffer.String()))
	return outputBuffer.String()[:len(outputBuffer.String())-len(token)]
}

func (o *TCPClient) ReadSize(size int) string {
	readSize := 0
	inputBuffer := make([]byte, 1)
	var outputBuffer bytes.Buffer
	for {
		o.ReadLock.Lock()
		n, err := o.Conn.Read(inputBuffer)
		o.ReadLock.Unlock()
		if err != nil {
			log.Error("Read from client failed")
			o.Server.DeleteTCPClient(o)
			break
		}
		// If read size equals zero, then finish reading
		outputBuffer.Write(inputBuffer[:n])
		readSize += n
		if readSize >= size {
			break
		}
	}
	log.Debug("(%d/%d) bytes read from client", len(outputBuffer.String()), size)
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
	log.Debug("%d bytes sent to client", n)
	return n
}

func (o *TCPClient) ParseHTTPRequest() {
	var err error
	// Request-Line
	o.Request.Method = o.ReadUntilClean(" ")
	urlString := o.ReadUntilClean(" ")
	o.Request.RequestURI, err = url.Parse(urlString)
	if err != nil {
		log.Error("Invalid url: %s", urlString)
		o.ResponseAndAbort("Invalid url")
		return
	}
	o.Request.HTTPVersion = o.ReadUntilClean("\r\n")
	log.Data("Method: %s (%d)", o.Request.Method, len(o.Request.Method))
	log.Data("RequestURI: %s", o.Request.RequestURI)
	log.Data("HTTPVersion: %s", o.Request.HTTPVersion)

	// Headers
	for {
		var line = o.ReadUntilClean("\r\n")
		// End of headers
		if line == "" {
			log.Debug("All header read")
			break
		}
		pair := strings.Split(line, ": ")
		log.Data("%s: %s", pair[0], pair[1])
		o.Request.Headers[pair[0]] = pair[1]
	}
	// Body
	if o.Request.Method == "POST" {
		contentLength, err := strconv.Atoi(o.Request.Headers["Content-Length"])
		if err != nil {
			o.ResponseAndAbort("Invalid Content-Length")
			return
		}
		o.Request.Body = o.ReadSize(contentLength)
		log.Data("Body: %s", o.Request.Body)
	} else {
		o.Request.Body = ""
	}
}

func (o *TCPClient) ParseHTTPResponse(response *HTTPResponse) {
	// Declare variables
	var err error
	// Status-Line
	response.HTTPVersion = o.ReadUntilClean(" ")
	statusCodeString := o.ReadUntilClean(" ")
	response.StatusCode, err = strconv.Atoi(statusCodeString)
	if err != nil {
		log.Error("Invalid status code: %s", err)
		// SHOULD NOT abort connection between client
	}
	response.ReasonPhrase = o.ReadUntilClean("\r\n")

	log.Data("HTTPVersion: %s", response.HTTPVersion)
	log.Data("StatusCode: %d", response.StatusCode)
	log.Data("ReasonPhrase: %s", response.ReasonPhrase)

	// Headers
	for {
		var line = o.ReadUntilClean("\r\n")
		// End of headers
		if line == "" {
			log.Debug("All header read")
			break
		}
		pair := strings.Split(line, ": ")
		log.Data("%s: %s", pair[0], pair[1])
		response.Headers[pair[0]] = pair[1]
	}
	log.Data("Headers: \n\t%s", response.Headers)

	// Body
	contentLength, err := strconv.Atoi(response.Headers["Content-Length"])
	if err != nil {
		log.Error("Invalid Content-Length: %s", err)
		// SHOULD NOT abort connection between client
	}
	response.Body = o.ReadSize(contentLength)
}

// Only support HTTP/1.0
// Methods:
//   HEAD/GET/POST
func (o *TCPClient) PrGoxy() {
	// Client guard
	if o.ClientFilterHandler() {
		return
	}
	// Parse HTTP Request
	o.ParseHTTPRequest()
	// Website guard
	if o.SiteFilterHandler() {
		return
	}
	// Redirect handler
	o.RedirectHandler()
	// Cache handler
	o.CacheHandler()
	// Proxy handler
	o.ProxyHandler()
}

func (o *TCPClient) ClientFilterHandler() bool {
	for _, v := range config.Cfg.Block.Hosts {
		// check if host:port starts with host
		if strings.HasPrefix(o.Conn.RemoteAddr().String(), v) {
			// blocked
			log.Warn("Client (%s) is blocked", v)
			o.ResponseAndAbort("Your IP is blocked")
			return true
		}
	}
	return false
}

func (o *TCPClient) SiteFilterHandler() bool {
	for _, v := range config.Cfg.Block.Sites {
		// Check hostname is blocked, without any port number
		if o.Request.RequestURI.Hostname() == v {
			// blocked
			log.Warn("Website (%s) is blocked", v)
			o.ResponseAndAbort("This website is blocked")
			return true
		}
	}
	return false
}

func (o *TCPClient) RedirectHandler() {

}

func (o *TCPClient) CacheHandler() {

}

func BuildHTTPRequest(request *HTTPRequest) string {
	// Convert Absolute URI -> Rel URI
	pathBuffer := new(bytes.Buffer)
	if request.RequestURI.Path != "" {
		pathBuffer.Write([]byte(request.RequestURI.Path))
	}
	if request.RequestURI.RawQuery != "" {
		pathBuffer.Write([]byte("?"))
		pathBuffer.Write([]byte(request.RequestURI.RawQuery))
	}
	if request.RequestURI.Fragment != "" {
		pathBuffer.Write([]byte("#"))
		pathBuffer.Write([]byte(request.RequestURI.Fragment))
	}
	// Rebuild request
	buffer := new(bytes.Buffer)
	buffer.WriteString(request.Method)
	buffer.WriteString(" ")
	buffer.WriteString(pathBuffer.String()) // TODO check URI hash
	buffer.WriteString(" ")
	buffer.WriteString(request.HTTPVersion)
	buffer.WriteString("\r\n")
	for k, v := range request.Headers {
		buffer.WriteString(k)
		buffer.WriteString(": ")
		buffer.WriteString(v)
		buffer.WriteString("\r\n")
	}
	buffer.WriteString("\r\n")
	buffer.WriteString(request.Body)

	return buffer.String()
}

func BuildHTTPResponse(response *HTTPResponse) string {
	// Status-Line
	buffer := new(bytes.Buffer)
	buffer.WriteString(response.HTTPVersion)
	buffer.WriteString(" ")
	buffer.WriteString(fmt.Sprintf("%d", response.StatusCode))
	buffer.WriteString(" ")
	buffer.WriteString(response.ReasonPhrase)
	buffer.WriteString("\r\n")
	// Headers
	for k, v := range response.Headers {
		buffer.WriteString(k)
		buffer.WriteString(": ")
		buffer.WriteString(v)
		buffer.WriteString("\r\n")
	}
	buffer.WriteString("\r\n")
	// Body
	buffer.WriteString(response.Body)
	return buffer.String()
}

func (o *TCPClient) ProxyHandler() {
	// Declare variables
	var err error
	// Construct HTTP Request
	requestData := BuildHTTPRequest(o.Request)
	log.Data("New request: \n%s", requestData)
	// Connect to server
	var port int
	port, err = strconv.Atoi(o.Request.RequestURI.Port())
	if err != nil {
		port = 80
	}
	target := fmt.Sprintf("%s:%d",
		o.Request.RequestURI.Hostname(),
		port,
	)
	conn, err := net.Dial(
		"tcp",
		target,
	)
	if err != nil {
		log.Error("Server (%s) is unavailable", target)
		o.ResponseAndAbort("Server is unavailable")
		return
	}
	client := CreateTCPClient(conn, o.Server)
	o.Server.AddTCPClient(client)
	// Send request to server
	client.Write([]byte(requestData))
	// Parse server response
	response := &HTTPResponse{
		Headers: make(map[string]string),
	}
	client.ParseHTTPResponse(response)
	// Build response
	responseData := BuildHTTPResponse(response)
	// Send response data to client
	log.Success("Response: \n%s", responseData)
	o.ResponseAndAbort(responseData)
	// Log
	log.Success("%s %s [%d][%d]", o.ToString(), o.Request.RequestURI, response.StatusCode, len(responseData))
}
