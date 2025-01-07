package gohttp

import (
	"errors"
	"fmt"
	"net"
)

func GET(request HTTPRequest) (HTTPResponse, error) {
	request.method = MethodGet
	tcpConn, err := net.Dial("tcp", request.uri.Host)
	if err != nil {
		return HTTPResponse{}, err
	}
	defer tcpConn.Close()

	buffer := make([]byte, 0, 1024)
	tcpConn.Write(request.toBytes())

	read, _ := tcpConn.Read(buffer)
	var response = parseResponsefromBytes(buffer[:read])
	return response, nil
}

func POST(request HTTPRequest) (HTTPResponse, error) {
	request.method = MethodPost
	if request.uri.Host == "" {
		host, ok := request.headers["host"]
		if !ok {
			return HTTPResponse{}, errors.New("no host to send request. Use absolute URI or host header")
		}
		request.uri.Host = host
	}
	tcpConn, err := net.Dial("tcp", request.uri.Host)
	if err != nil {
		return HTTPResponse{}, err
	}
	defer tcpConn.Close()

	_, err = tcpConn.Write(request.toBytes())
	if err != nil {
		fmt.Println(err.Error())
	}

	var buffer []byte = make([]byte, 1024)
	read, _ := tcpConn.Read(buffer)
	var response = parseResponsefromBytes(buffer[:read])
	return response, nil
}
