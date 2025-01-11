package gohttp

import (
	"bytes"
	"fmt"
	"net"
)

type HTTPServer struct {
	address     string
	listener    net.Listener
	uriHandlers map[string][]*responseHandlers
}

type ResponseFunction func(HTTPRequest, *HTTPResponseWriter)

type responseHandlers struct {
	uriPattern string
	handler    ResponseFunction
}

func (s HTTPServer) HandleGET(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandlers = new(responseHandlers)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction

	if currentHandlers, exists := s.uriHandlers["GET"]; exists {
		s.uriHandlers["GET"] = append(currentHandlers, handler)
	} else {
		currentHandlers = make([]*responseHandlers, 0)
		s.uriHandlers["GET"] = append(currentHandlers, handler)
	}
}

func (s HTTPServer) HandleRequest() error {
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
				statusCode: STATUS_OK,
				buffer:     new(bytes.Buffer),
			}

			if request.method == "GET" {
				if handlers, exists := s.uriHandlers["GET"]; exists {
					var handled = false
					for _, handler := range handlers {
						var uriPattern = handler.uriPattern
						if isURIMatch(request.uri.Path, uriPattern) {
							handler.handler(*request, responseWriter)
							handled = true
							break
						}
					}
					if !handled {
						responseWriter.statusCode = STATUS_NOT_IMPLEMENTED
					}
				} else {
					responseWriter.statusCode = STATUS_NOT_IMPLEMENTED
				}
			}
			var response = newHTTPResponse(*responseWriter)

			responseBytes, err := response.toBytes()
			if err != nil {
				break
			}
			connection.Write(responseBytes)
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
		address:     address,
		listener:    listener,
		uriHandlers: make(map[string][]*responseHandlers),
	}, nil
}
