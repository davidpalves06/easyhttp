package easyhttp

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

type ClientHTTPRequest struct {
	method          string
	uri             *url.URL
	version         string
	headers         Headers
	cookies         []*Cookie
	body            []byte
	chunkChannel    chan []byte
	chunked         bool
	onResponseChunk ClientChunkFunction
	timeout         time.Duration
}

func (r *ClientHTTPRequest) SetHeader(key string, value string) {
	r.headers[strings.ToLower(strings.TrimSpace(key))] = []string{strings.TrimSpace(value)}
}

func (r *ClientHTTPRequest) CloseConnection() {
	r.SetHeader("Connection", "close")
}

func (r *ClientHTTPRequest) SetTimeout(timeout_ms time.Duration) {
	r.timeout = timeout_ms
}

func (r *ClientHTTPRequest) AddHeader(key string, value string) {
	headers, exists := r.headers[strings.ToLower(strings.TrimSpace(key))]
	if !exists {
		headers = []string{}
	}
	headers = append(headers, value)
	r.headers[strings.ToLower(strings.TrimSpace(key))] = headers
}

func (r *ClientHTTPRequest) GetHeader(key string) []string {
	value, found := r.headers[strings.ToLower(key)]
	if found {
		return value
	} else {
		return nil
	}
}

func (r *ClientHTTPRequest) HasHeaderValue(key string, value string) bool {
	headers, found := r.headers[strings.ToLower(key)]
	if found && slices.Contains(headers, value) {
		return true
	} else {
		return false
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
		builder := new(strings.Builder)
		builder.WriteString(headerName)
		builder.WriteString(": ")
		for i, value := range headerValue {
			builder.WriteString(value)
			if i < len(headerValue)-1 {
				builder.WriteString(", ")
			}
		}
		builder.WriteString("\r\n")
		buffer.WriteString(builder.String())
	}

	if len(r.cookies) > 0 {
		cookieBuilder := new(strings.Builder)
		cookieBuilder.WriteString("Cookie: ")
		for i, cookie := range r.cookies {
			cookieBuilder.WriteString(cookie.Name)
			cookieBuilder.WriteString("=")
			cookieBuilder.WriteString(cookie.Value)
			if i < len(r.cookies)-1 {
				cookieBuilder.WriteString("; ")
			}
		}
		cookieBuilder.WriteString("\r\n")
		buffer.WriteString(cookieBuilder.String())
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
		cookies:      make([]*Cookie, 0, 5),
	}

	if len(body) > 0 {
		newRequest.SetHeader("Content-Length", strconv.Itoa(len(body)))
		newRequest.SetHeader("Content-Type", "text/plain")
	}

	newRequest.SetHeader("User-Agent", softwareName)

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
		cookies:      make([]*Cookie, 0, 5),
	}

	newRequest.SetHeader("User-Agent", softwareName)

	return newRequest, nil
}
