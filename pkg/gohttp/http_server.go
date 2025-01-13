package gohttp

import (
	"bytes"
	"log"
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

type ResponseFunction func(HTTPRequest, *HTTPResponseWriter)

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
}

func (s *HTTPServer) HandlePOST(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandlers = new(responseHandlers)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction

	s.addHandlerForMethod(handler, MethodPost)
}

func HandleConnection(connection net.Conn, server *HTTPServer) {
	var buffer []byte = make([]byte, 2048)
	defer connection.Close()
	defer server.waitGroup.Done()
	// for server.running {
	bytesRead, err := connection.Read(buffer)
	if err != nil || bytesRead == 0 {
		// break
		return
	}

	request, err := parseRequestFromBytes(buffer, bytesRead)
	if err != nil {
		log.Println(err.Error())
		// break
		return
	}

	responseWriter := &HTTPResponseWriter{
		headers:    make(map[string]string),
		statusCode: STATUS_OK,
		buffer:     new(bytes.Buffer),
	}

	if handlers, exists := server.uriHandlers[request.method]; exists {
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
	var response = newHTTPResponse(*responseWriter)
	response.version = request.version

	responseBytes, err := response.toBytes()
	if err != nil {
		// break
		return
	}
	connection.Write(responseBytes)
	// }
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
