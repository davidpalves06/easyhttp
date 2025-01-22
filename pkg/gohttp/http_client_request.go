package gohttp

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

type ClientHTTPRequest struct {
	method          string
	uri             *url.URL
	version         string
	headers         Headers
	body            []byte
	chunkChannel    chan []byte
	chunked         bool
	onResponseChunk ClientChunkFunction
}

func (r *ClientHTTPRequest) SetHeader(key string, value string) {
	r.headers[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
}

func (r *ClientHTTPRequest) GetHeader(key string) string {
	value, found := r.headers[strings.ToLower(key)]
	if found {
		return value
	} else {
		return ""
	}
}

func (r *ClientHTTPRequest) Headers() Headers {
	return r.headers
}

func (r *ClientHTTPRequest) Version() string {
	return r.version
}

func (r *ClientHTTPRequest) SetVersion(version string) error {
	if slices.Contains(validVersions, version) {
		r.version = version
		return nil
	}
	return errors.New("invalid Version")
}

func (r *ClientHTTPRequest) SetBody(body []byte) {
	r.body = body
}

func (r *ClientHTTPRequest) SetURI(uri string) error {
	requestURI, err := url.ParseRequestURI(uri)
	if err != nil {
		return errors.New("uri is not valid")
	}
	r.uri = requestURI
	return nil
}

func (r *ClientHTTPRequest) SendChunk(chunk []byte) {
	r.chunkChannel <- chunk
}

func (r *ClientHTTPRequest) Done() {
	close(r.chunkChannel)
}

func (r *ClientHTTPRequest) Chunked() {
	r.chunked = true
}

func (r *ClientHTTPRequest) OnChunkFunction(onChunk ClientChunkFunction) {
	r.onResponseChunk = onChunk
}

func (r ClientHTTPRequest) sendChunks(connection net.Conn) {
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

func (r ClientHTTPRequest) toBytes() ([]byte, error) {
	buffer := new(bytes.Buffer)
	var requestLine = fmt.Sprintf("%s %s HTTP/%s\r\n", r.method, r.uri.RequestURI(), r.version)
	buffer.WriteString(requestLine)

	if r.chunked {
		r.SetHeader("Transfer-Encoding", "chunked")
	} else if len(r.body) > 0 {
		r.SetHeader("Content-Length", strconv.Itoa(len(r.body)))
	}

	for headerName, headerValue := range r.headers {
		var headerLine = fmt.Sprintf("%s: %s\r\n", headerName, headerValue)
		buffer.WriteString(headerLine)
	}

	buffer.WriteString("\r\n")

	if r.body != nil && !r.chunked {
		bodyLength := len(r.body)
		if bodyLength == 0 {
			return nil, errors.New("content length is not valid")
		}

		if r.method == "GET" || r.method == "HEAD" {
			return nil, fmt.Errorf("method %s should not have a body", r.method)
		}

		buffer.Write(r.body[:bodyLength])
	}
	return buffer.Bytes(), nil
}

func NewRequestWithBody(uri string, body []byte) (ClientHTTPRequest, error) {
	requestURI, err := url.ParseRequestURI(uri)
	if err != nil {
		return ClientHTTPRequest{}, errors.New("uri is not valid")
	}

	newRequest := ClientHTTPRequest{
		headers:      make(Headers),
		version:      "1.1",
		body:         body,
		chunkChannel: make(chan []byte, 1),
		chunked:      false,
		uri:          requestURI,
	}

	if len(body) > 0 {
		newRequest.SetHeader("Content-Length", strconv.Itoa(len(body)))
		newRequest.SetHeader("Content-Type", "text/plain")
	}

	newRequest.headers["User-Agent"] = softwareName
	return newRequest, nil
}

func NewRequest(uri string) (ClientHTTPRequest, error) {
	requestURI, err := url.ParseRequestURI(uri)
	if err != nil {
		return ClientHTTPRequest{}, errors.New("uri is not valid")
	}
	newRequest := ClientHTTPRequest{
		headers:      make(Headers),
		version:      "1.1",
		body:         nil,
		chunkChannel: make(chan []byte, 1),
		chunked:      false,
		uri:          requestURI,
	}

	newRequest.headers["User-Agent"] = softwareName
	return newRequest, nil
}
