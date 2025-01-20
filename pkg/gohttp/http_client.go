package gohttp

import (
	"errors"
	"io"
	"net"
	"time"
)

var activeConnections map[string]net.Conn = make(map[string]net.Conn)

func GET(request HTTPRequest) (*HTTPResponse, error) {
	request.method = MethodGet
	return makeRequest(request)
}

func HEAD(request HTTPRequest) (*HTTPResponse, error) {
	request.method = MethodHead
	return makeRequest(request)
}

func POST(request HTTPRequest) (*HTTPResponse, error) {
	request.method = MethodPost
	return makeRequest(request)
}

func makeRequest(request HTTPRequest) (*HTTPResponse, error) {
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

	connection, exists := activeConnections[request.uri.Host]
	if !exists || !checkIfConnectionIsStillOpen(connection) {
		connection, err = net.Dial("tcp", request.uri.Host)
		if err != nil {
			return nil, err
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

	response, err := parseResponsefromConnection(connection)
	if err != nil {
		return nil, err
	}

	if isClosingRequest(&request) {
		connection.Close()
		delete(activeConnections, request.uri.Host)
	} else {
		activeConnections[request.uri.Host] = connection
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
