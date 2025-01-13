package gohttp

import (
	"errors"
	"net"
)

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
	tcpConn, err := net.Dial("tcp", request.uri.Host)
	if err != nil {
		return nil, err
	}
	defer tcpConn.Close()

	requestBytes, err := request.toBytes()
	if err != nil {
		return nil, err
	}
	_, err = tcpConn.Write(requestBytes)
	if err != nil {
		return nil, err
	}

	var buffer []byte = make([]byte, 1024)
	read, _ := tcpConn.Read(buffer)
	response, err := parseResponsefromBytes(buffer[:read])
	if err != nil {
		return nil, err
	}
	return response, nil
}
