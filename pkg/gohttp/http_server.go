package gohttp

import (
	"bytes"
	"net"
	"sync"
)

type HTTPServer struct {
	address     string
	listener    net.Listener
	uriHandlers map[string][]*responseHandlers
	running     bool
	waitGroup   sync.WaitGroup
}

type ResponseFunction func(HTTPRequest, *HTTPResponse)

type responseHandlers struct {
	uriPattern string
	handler    ResponseFunction
}

func (s *HTTPServer) addHandlerForMethod(handler *responseHandlers, method string) {

	if currentHandlers, exists := s.uriHandlers[method]; exists {
		s.uriHandlers[method] = append(currentHandlers, handler)
	} else {
		currentHandlers = make([]*responseHandlers, 0)
		s.uriHandlers[method] = append(currentHandlers, handler)
	}
}

func (s *HTTPServer) HandleGET(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandlers = new(responseHandlers)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction

	s.addHandlerForMethod(handler, MethodGet)
	s.addHandlerForMethod(handler, MethodHead)
}

func (s *HTTPServer) HandlePOST(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandlers = new(responseHandlers)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction

	s.addHandlerForMethod(handler, MethodPost)
}

func HandleConnection(connection net.Conn, server *HTTPServer) {
	defer connection.Close()
	defer server.waitGroup.Done()
	var keepAlive = true
	for server.running && keepAlive {
		request, err := parseRequestFromConnection(connection)
		if err != nil {
			if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
				badRequestResponse := newBadRequestResponse()
				responseBytes, _ := badRequestResponse.toBytes()
				connection.Write(responseBytes)
			}
			return
		}

		response := &HTTPResponse{
			headers:    make(map[string]string),
			statusCode: STATUS_OK,
			body:       new(bytes.Buffer),
			conn:       connection,
			version:    request.version,
			method:     request.method,
		}

		if handlers, exists := server.uriHandlers[request.method]; exists {
			var handled = false
			for _, handler := range handlers {
				var uriPattern = handler.uriPattern
				if isURIMatch(request.uri.Path, uriPattern) {
					handler.handler(*request, response)
					handled = true
					break
				}
			}
			if !handled {
				response.statusCode = STATUS_NOT_IMPLEMENTED
			}
		} else {
			response.statusCode = STATUS_NOT_IMPLEMENTED
		}

		if request.method == MethodHead {
			response.body = nil
		}

		if !response.chunked {
			responseBytes, err := response.toBytes()
			if err != nil {
				badRequestResponse := newBadRequestResponse()
				responseBytes, _ := badRequestResponse.toBytes()
				connection.Write(responseBytes)
				return
			}
			connection.Write(responseBytes)
		} else {
			connection.Write([]byte("0 \r\n\r\n"))
		}
		keepAlive = !isClosingRequest(request)
	}
}

func (s *HTTPServer) AcceptConnection() (net.Conn, error) {
	return s.listener.Accept()
}

func (s *HTTPServer) Run() {
	s.running = true
	for s.running {
		connection, err := s.AcceptConnection()
		if err != nil {
			break
		}
		s.waitGroup.Add(1)
		go HandleConnection(connection, s)
	}
}

func (s *HTTPServer) Close() error {
	s.running = false
	err := s.listener.Close()
	s.waitGroup.Wait()
	return err
}

func NewHTTPServer(address string) (*HTTPServer, error) {
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
