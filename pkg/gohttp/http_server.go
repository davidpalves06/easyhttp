package gohttp

import (
	"bytes"
	"fmt"
	"net"
)

type HTTPServer struct {
	address  string
	listener net.Listener
}

func (s HTTPServer) HandleRequest(fn func(HTTPRequest, *HTTPResponseWriter)) error {
	connection, err := s.listener.Accept()
	if err != nil {
		return err
	}
	go func() {
		var buffer []byte = make([]byte, 2048)
		defer connection.Close()
		for {
			bytesRead, err := connection.Read(buffer)
			if err != nil || bytesRead == 0 {
				break
			}

			request, err := parseRequestFromBytes(buffer, bytesRead)
			if err != nil {
				fmt.Println(err.Error())
				break
			}

			responseWriter := &HTTPResponseWriter{
				headers:    make(map[string]string),
				statusCode: 200,
				buffer:     new(bytes.Buffer),
			}

			fn(*request, responseWriter)

			var response = newHTTPResponse(*responseWriter)

			connection.Write(response.toBytes())
		}
	}()

	return nil
}

func (s HTTPServer) Close() error {
	err := s.listener.Close()
	return err
}

func CreateHTTPServer(address string) (*HTTPServer, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return &HTTPServer{
		address:  address,
		listener: listener,
	}, nil
}
