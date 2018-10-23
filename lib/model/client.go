package model

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
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

var Cache map[string]HTTPResponse

func init() {
	if Cache == nil {
		Cache = map[string]HTTPResponse{}
	}
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
	if size <= 0 {
		return ""
	}
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
		delimiter := ":"
		index := strings.Index(line, delimiter)
		headerKey := line[:index]
		headerValue := LeftStrip(line[index+len(delimiter):])
		o.Request.Headers[headerKey] = headerValue
	}

	log.Data("Request Headers: \n\t%s", o.Request.Headers)

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
	log.Data("Request Body: \n\t%s", o.Request.Body)
}

func LeftStrip(data string) string {
	var k int = 0
	var v rune
	for k, v = range data {
		// Check space
		if v != '\x20' {
			break
		}
	}
	return data[k:]
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
		delimiter := ":"
		index := strings.Index(line, delimiter)
		headerKey := line[:index]
		headerValue := LeftStrip(line[index+len(delimiter):])
		response.Headers[headerKey] = headerValue
	}
	log.Data("Response Headers: \n\t%s", response.Headers)

	// Body
	contentLength, err := strconv.Atoi(response.Headers["Content-Length"])
	if err != nil {
		log.Error("Invalid Content-Length: %s", err)
		// SHOULD NOT abort connection between client
	}
	if contentLength > 0 {
		response.Body = o.ReadSize(contentLength)

	} else {
		response.Body = ""
	}
}

func Pipe(in *TCPClient, out *TCPClient, desc string) {
	defer in.ResponseAndAbort("")
	defer out.ResponseAndAbort("")
	var buffer = make([]byte, 0x4000)
	for {
		n, err := in.Conn.Read(buffer)
		if err != nil {
			log.Debug("[%s] Unable to read from input, error: %s\n", desc, err.Error())
			break
		}
		n, err = out.Conn.Write(buffer[:n])
		if err != nil {
			log.Debug("[%s] Unable to write to output, error: %s\n", desc, err.Error())
			break
		}
	}
}

func (o *TCPClient) HTTPTunnel() {
	host := GetHostname(o.Request.RequestURI.String())
	port := GetPort(o.Request.RequestURI.String(), 443)
	log.Debug("Handling HTTP tunnel to %s:%d", host, port)
	client := ProxyConnectToServer(o, host, port)
	if client == nil {
		log.Error("Server (%s:%d) is unavailable", host, port)
		o.ResponseAndAbort("Server is unavailable")
		return
	}
	log.Success("Connection to %s:%d established", host, port)
	// HTTP/1.1 200 Connection established
	response := &HTTPResponse{
		HTTPVersion:  "HTTP/1.1",
		StatusCode:   200,
		ReasonPhrase: "Connection established",
	}
	o.Write([]byte(BuildHTTPResponse(response)))
	// Transfer data
	go Pipe(client, o, "Server -> Client")
	go Pipe(o, client, "Client -> Server")
}

// Only support HTTP/1.0
// Methods:
//   HEAD/GET/POST
func (o *TCPClient) PrGoxy() {
	// Client guard
	if o.ClientFilterHandler() {
		return
	}
	o.ParseHTTPRequest()
	// Website guard
	if o.SiteFilterHandler() {
		return
	}
	// Redirect handler
	o.RedirectHandler()
	// Support for HTTP Tunnel
	if o.Request.Method == "CONNECT" {
		o.HTTPTunnel()
		return
	}
	// Cache handler
	if config.Cfg.Cache && o.CacheHandler() {
		return
	}
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

func GetHostname(host string) string {
	return strings.Split(host, ":")[0]
}

func GetPort(host string, default_port int16) int16 {
	pair := strings.Split(host, ":")
	if len(pair) < 2 {
		return default_port
	}
	port, err := strconv.Atoi(pair[len(pair)-1])
	if err != nil {
		return default_port
	}
	return int16(port)
}

func (o *TCPClient) RedirectHandler() {
	// Parse port in Request-URI
	// Check redirect table
	for k, v := range config.Cfg.Redirect {
		srcHostname := GetHostname(o.Request.RequestURI.Host)
		srcPort := GetPort(o.Request.RequestURI.Host, 80)
		dstHostname := GetHostname(k)
		dstPort := GetPort(k, 80)
		targetHostname := GetHostname(v)
		targetPort := GetPort(v, 80)
		log.Debug("src: %s:%d", srcHostname, srcPort)
		log.Debug("dst: %s:%d", dstHostname, dstPort)
		log.Debug("target: %s:%d", targetHostname, targetPort)
		if srcHostname == dstHostname && srcPort == dstPort {
			log.Success("Redirect %s => %s", k, v)
			target := fmt.Sprintf(
				"%s:%d",
				targetHostname,
				targetPort,
			)
			// Change RequestURI
			o.Request.RequestURI.Host = target
			// Change Host
			o.Request.Headers["Host"] = target
		}
	}
}

func Cachable(request *HTTPRequest) bool {
	return (request.Method == "GET" || request.Method == "HEAD") && request.Headers["Range"] == ""
}

func CacheHit(uri string) (bool, HTTPResponse) {
	for k, v := range Cache {
		if k == uri {
			return true, v
		}
	}
	return false, HTTPResponse{}
}

// func IfModifiedSince(request HTTPRequest, lastModified string) {}

func (o *TCPClient) CacheHandler() bool {
	if !Cachable(o.Request) {
		return false
	}
	var err error
	if ok, response := CacheHit(o.Request.RequestURI.String()); ok {
		// Send If-Modify-Since
		ifModifySince := time.Time{}
		if v, ok := response.Headers["Last-Modified"]; ok {
			ifModifySince, err = http.ParseTime(v)
			if err != nil {
				log.Debug("Failed to parse time, %s", err)
			}
		} else if v, ok := response.Headers["Date"]; ok {
			ifModifySince, err = http.ParseTime(v)
			if err != nil {
				log.Debug("Failed to parse time, %s", err)
			}
		} else {
			ifModifySince = time.Now()
		}
		// Connect to server
		host := GetHostname(o.Request.RequestURI.Host)
		port := GetPort(o.Request.RequestURI.Host, 80)
		client := ProxyConnectToServer(o, host, port)
		if client == nil {
			// false represents that proxy handler does not handle the request
			return false
		}

		o.Request.Headers["If-Modified-Since"] = ifModifySince.Format(time.RFC1123)
		client.Write([]byte(BuildHTTPRequest(o.Request)))
		ifModifySinceResponse := &HTTPResponse{
			Headers: make(map[string]string),
		}
		client.ParseHTTPResponse(ifModifySinceResponse)
		// If 304 Not Modified
		//     Send cache
		// Else
		//     Save to cache
		statusCode := ifModifySinceResponse.StatusCode
		if statusCode == 304 {
			responseData := BuildHTTPResponse(&response)
			o.ResponseAndAbort(responseData)
			log.Success("%s %s %s [CACHE][%d]", o.Request.Method, o.ToString(), o.Request.RequestURI, len(responseData))
			log.Success("304 Not modified")
		} else {
			// Need refresh cache
			responseData := BuildHTTPResponse(ifModifySinceResponse)
			o.ResponseAndAbort(responseData)
			log.Success("%s %s %s [CACHE][%d]", o.Request.Method, o.ToString(), o.Request.RequestURI, len(responseData))
			// refresh cache
			log.Success("Renovating cache")
			Cache[o.Request.RequestURI.String()] = *ifModifySinceResponse
		}
		return true
	}
	return false
}

func ProxyConnectToServer(o *TCPClient, host string, port int16) *TCPClient {
	var err error
	target := fmt.Sprintf("%s:%d",
		host,
		port,
	)
	log.Debug("Connecting to %s", target)
	conn, err := net.Dial(
		"tcp",
		target,
	)
	if err != nil {
		return nil
	}
	client := CreateTCPClient(conn, o.Server)
	o.Server.AddTCPClient(client)
	return client
}

func (o *TCPClient) ProxyHandler() {
	// Construct HTTP Request
	// Force HTTP/1.0
	o.Request.HTTPVersion = "HTTP/1.0"
	requestData := BuildHTTPRequest(o.Request)
	log.Data("Rewrited Request: \n%s", requestData)
	// Connect to server
	host := GetHostname(o.Request.RequestURI.Host)
	port := GetPort(o.Request.RequestURI.Host, 80)

	client := ProxyConnectToServer(o, host, port)
	if client == nil {
		log.Error("Server (%s:%d) is unavailable", host, port)
		o.ResponseAndAbort("Server is unavailable")
		return
	}
	// Send request to server
	client.Write([]byte(requestData))
	// Parse server response
	response := &HTTPResponse{
		Headers: make(map[string]string),
	}
	client.ParseHTTPResponse(response)
	// Build response
	responseData := BuildHTTPResponse(response)
	log.Data(responseData)
	// Send response data to client
	o.ResponseAndAbort(responseData)

	// Cache
	Cache[o.Request.RequestURI.String()] = *response

	// Log
	log.Success("%s %s %s [%d][%d]", o.Request.Method, o.ToString(), o.Request.RequestURI, response.StatusCode, len(responseData))
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
