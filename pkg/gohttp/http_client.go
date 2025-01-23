package gohttp

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/textproto"
	"time"
)

type httpClient struct {
	activeConnections map[string]net.Conn
	TLSConfig         *tls.Config
}

func NewHTTPClient() httpClient {
	return httpClient{
		activeConnections: make(map[string]net.Conn),
	}
}

func (c *httpClient) GET(request ClientHTTPRequest) (*ClientHTTPResponse, error) {
	request.method = MethodGet
	return c.sendRequest(request)
}

func (c *httpClient) HEAD(request ClientHTTPRequest) (*ClientHTTPResponse, error) {
	request.method = MethodHead
	return c.sendRequest(request)
}

func (c *httpClient) POST(request ClientHTTPRequest) (*ClientHTTPResponse, error) {
	request.method = MethodPost
	return c.sendRequest(request)
}

func (c *httpClient) sendRequest(request ClientHTTPRequest) (*ClientHTTPResponse, error) {
	if request.uri.Host == "" {
		host, ok := request.headers["host"]
		if !ok {
			return nil, errors.New("no host to send request. Use absolute URI or host header")
		}
		request.uri.Host = host
	}

	request.SetHeader("Host", request.uri.Host)

	var connection net.Conn
	var err error

	connection, exists := c.activeConnections[request.uri.Host]
	if !exists || !checkIfConnectionIsStillOpen(connection) {
		if request.uri.Scheme == "https" {
			connection, err = tls.Dial("tcp", request.uri.Host, c.TLSConfig)
			if err != nil {
				return nil, err
			}
		} else {
			connection, err = net.Dial("tcp", request.uri.Host)
			if err != nil {
				return nil, err
			}
		}
	}

	requestBytes, err := request.toBytes()
	if err != nil {
		return nil, err
	}
	_, err = connection.Write(requestBytes)
	if err != nil {
		return nil, err
	}

	if request.chunked {
		request.sendChunks(connection)
	}

	var responseReader = textproto.NewReader(bufio.NewReader(connection))
	response, err := parseResponsefromConnection(responseReader)
	if err != nil {
		return nil, err
	}

	err = parseResponseBody(response, connection, responseReader, request.onResponseChunk)
	if err != nil {
		return nil, err
	}

	if isClosingRequest(&request) {
		connection.Close()
		delete(c.activeConnections, request.uri.Host)
	} else {
		c.activeConnections[request.uri.Host] = connection
	}
	return response, nil
}

func checkIfConnectionIsStillOpen(connection net.Conn) bool {
	one := make([]byte, 1)
	connection.SetReadDeadline(time.Now().Add(100 * time.Microsecond))

	if _, err := connection.Read(one); err == io.EOF {
		return false
	} else {
		connection.SetReadDeadline(time.Time{})
		return true
	}
}
