package gohttp

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

type HTTPRequest struct {
	method          string
	uri             *url.URL
	version         string
	headers         Headers
	Body            []byte
	chunkChannel    chan []byte
	chunked         bool
	connection      *net.Conn
	onResponseChunk ClientChunkFunction
}

func (r *HTTPRequest) SetHeader(key string, value string) {
	r.headers[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
}

func (r *HTTPRequest) GetHeader(key string) string {
	value, found := r.headers[strings.ToLower(key)]
	if found {
		return value
	} else {
		return ""
	}
}

func (r *HTTPRequest) ExistsHeader(key string) bool {
	_, found := r.headers[strings.ToLower(key)]
	return found
}

func (r *HTTPRequest) Headers() Headers {
	return r.headers
}

func (r *HTTPRequest) Version(version string) error {
	if slices.Contains(validVersions, version) {
		r.version = version
		return nil
	}
	return errors.New("invalid Version")
}

func (r *HTTPRequest) SendChunk(chunk []byte) {
	r.chunkChannel <- chunk
}

func (r *HTTPRequest) Done() {
	close(r.chunkChannel)
}

func (r *HTTPRequest) Chunked() {
	r.chunked = true
}

func (r HTTPRequest) toBytes() ([]byte, error) {
	buffer := new(bytes.Buffer)
	var requestLine = fmt.Sprintf("%s %s HTTP/%s\r\n", r.method, r.uri.RequestURI(), r.version)
	buffer.WriteString(requestLine)

	if r.chunked {
		r.SetHeader("Transfer-Encoding", "chunked")
	} else if len(r.Body) > 0 {
		r.SetHeader("Content-Length", strconv.Itoa(len(r.Body)))
	}

	r.headers["User-Agent"] = softwareName

	for headerName, headerValue := range r.headers {
		var headerLine = fmt.Sprintf("%s: %s\r\n", headerName, headerValue)
		buffer.WriteString(headerLine)
	}

	buffer.WriteString("\r\n")

	if r.Body != nil && !r.chunked {
		bodyLength := len(r.Body)
		if bodyLength == 0 {
			return nil, errors.New("content length is not valid")
		}

		if r.method == "GET" || r.method == "HEAD" {
			return nil, fmt.Errorf("method %s should not have a body", r.method)
		}

		buffer.Write(r.Body[:bodyLength])
	}
	return buffer.Bytes(), nil
}

func (r HTTPRequest) sendChunks(connection net.Conn) {
	for chunk := range r.chunkChannel {
		buffer := new(bytes.Buffer)
		chunkLength := fmt.Sprintf("%x \r\n", len(chunk))
		buffer.WriteString(chunkLength)
		buffer.Write(chunk)

		buffer.WriteString("\r\n")

		connection.Write(buffer.Bytes())
	}

	connection.Write([]byte("0 \r\n\r\n"))
}

func NewRequestWithBody(uri string, body []byte) (HTTPRequest, error) {
	requestURI, err := url.ParseRequestURI(uri)
	if err != nil {
		return HTTPRequest{}, errors.New("uri is not valid")
	}

	newRequest := HTTPRequest{
		headers:      make(Headers),
		version:      "1.1",
		Body:         body,
		chunkChannel: make(chan []byte, 1),
		chunked:      false,
		uri:          requestURI,
	}

	if len(body) > 0 {
		newRequest.SetHeader("Content-Length", strconv.Itoa(len(body)))
		newRequest.SetHeader("Content-Type", "text/plain")
	}
	return newRequest, nil
}

func NewRequest(uri string) (HTTPRequest, error) {
	requestURI, err := url.ParseRequestURI(uri)
	if err != nil {
		return HTTPRequest{}, errors.New("uri is not valid")
	}
	newRequest := HTTPRequest{
		headers:      make(Headers),
		version:      "1.1",
		Body:         nil,
		chunkChannel: make(chan []byte, 1),
		chunked:      false,
		uri:          requestURI,
	}
	return newRequest, nil
}

func parseRequestLine(requestLine string, request *HTTPRequest) error {
	var requestLineSplit = strings.Split(requestLine, " ")
	if len(requestLineSplit) != 3 {
		return ErrParsing
	}
	var method string = requestLineSplit[0]
	if !slices.Contains(validMethods, method) {
		return ErrParsing
	}
	request.method = method

	var requestUri = requestLineSplit[1]
	parsedUri, err := url.ParseRequestURI(requestUri)
	if err != nil {
		return ErrParsing
	}
	request.uri = parsedUri

	var version = requestLineSplit[2]
	versionSplit := strings.Split(version, "/")
	if len(versionSplit) != 2 || versionSplit[0] != "HTTP" || !slices.Contains(validVersions, versionSplit[1]) {
		return ErrParsing
	}
	request.version = versionSplit[1]
	return nil

}

func parseHeaders(requestReader *textproto.Reader, request *HTTPRequest) {
	for {
		var line, err = requestReader.ReadLine()
		if err != nil {
			continue
		}
		if line == "" {
			break
		}
		headerSplit := strings.Split(line, ":")
		if len(headerSplit) >= 2 {
			request.SetHeader(headerSplit[0], strings.Join(headerSplit[1:], ":"))
		}
	}
}

func parseRequestFromConnection(requestReader *textproto.Reader, connection net.Conn) (*HTTPRequest, error) {
	connection.SetReadDeadline(time.Now().Add(KEEP_ALIVE_TIMEOUT * time.Second))
	var request *HTTPRequest = &HTTPRequest{
		headers:    make(map[string]string),
		connection: &connection,
	}
	requestLine, err := requestReader.ReadLine()
	if err != nil {
		return nil, err
	}
	err = parseRequestLine(requestLine, request)
	if err != nil {
		return nil, err
	}

	parseHeaders(requestReader, request)
	if request.GetHeader("Host") == "" {
		return nil, ErrParsing
	}

	return request, nil
}

func isClosingRequest(request *HTTPRequest) bool {
	connection := request.GetHeader("Connection")
	if request.version == "1.0" {
		return connection != "keep-alive"
	} else {
		return connection == "close"
	}
}
