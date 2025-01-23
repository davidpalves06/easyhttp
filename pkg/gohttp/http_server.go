package gohttp

import (
	"bufio"
	"crypto/tls"
	"errors"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"time"
)

type HTTPServer struct {
	address     string
	listener    net.Listener
	uriHandlers map[string][]*responseHandlers
	running     bool
	waitGroup   sync.WaitGroup
}

type ResponseFunction func(ServerHTTPRequest, *ServerHTTPResponse)
type ServerChunkFunction func([]byte, ServerHTTPRequest, *ServerHTTPResponse) bool
type ClientChunkFunction func([]byte, *ClientHTTPResponse) bool

type HandlerOptions struct {
	onChunk        ServerChunkFunction
	runAfterChunks bool
}

type responseHandlers struct {
	uriPattern string
	handler    ResponseFunction
	options    HandlerOptions
}

func FileServer(filePrefix string) ResponseFunction {
	return func(request ServerHTTPRequest, response *ServerHTTPResponse) {
		response.statusCode = STATUS_OK
		var requestPath string = request.Path()
		splittedPath := strings.Split(requestPath, "/")
		filePrefix, _ = strings.CutSuffix(filePrefix, "/")

		fileNameBuilder := new(strings.Builder)
		fileNameBuilder.WriteString(filePrefix)
		fileNameBuilder.WriteString("/")
		fileNameBuilder.WriteString(splittedPath[len(splittedPath)-1])

		response.SendFile(fileNameBuilder.String())
	}
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
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodGet)
	s.addHandlerForMethod(handler, MethodHead)
}

func (s *HTTPServer) HandleGETWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandlers = new(responseHandlers)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodGet)
	s.addHandlerForMethod(handler, MethodHead)
}

func (s *HTTPServer) HandlePOST(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandlers = new(responseHandlers)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodPost)
}

func (s *HTTPServer) HandlePOSTWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandlers = new(responseHandlers)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodPost)
}

func HandleConnection(connection net.Conn, server *HTTPServer) {
	defer connection.Close()
	defer server.waitGroup.Done()
	var keepAlive = true
	for server.running && keepAlive {
		var requestReader = textproto.NewReader(bufio.NewReader(connection))
		connection.SetReadDeadline(time.Now().Add(KEEP_ALIVE_TIMEOUT * time.Second))
		request, err := parseRequestFromConnection(requestReader)
		if err != nil {
			sendBadRequestIfNotTimeout(err, connection)
			return
		}

		response := newHTTPResponse(request, connection)

		handler, err := getRequestHandler(server, request)
		if err != nil {
			response.statusCode = STATUS_NOT_IMPLEMENTED
		} else {
			err := parseRequestBody(request, connection, requestReader, response, handler.options.onChunk)
			if err != nil {
				sendBadRequestIfNotTimeout(err, connection)
				return
			}

			if handler.options.onChunk == nil || handler.options.runAfterChunks {
				handler.handler(*request, response)
			}
		}

		if request.method == MethodHead {
			response.body = nil
		}

		if !response.chunked {
			responseBytes, err := response.toBytes()
			if err != nil {
				sendBadRequestIfNotTimeout(err, connection)
				return
			}
			connection.Write(responseBytes)
		} else {
			connection.Write([]byte("0 \r\n\r\n"))
		}
		keepAlive = !isClosingRequest(request)
	}
}

func sendBadRequestIfNotTimeout(err error, connection net.Conn) {
	if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		badRequestResponse := newBadRequestResponse()
		responseBytes, _ := badRequestResponse.toBytes()
		connection.Write(responseBytes)
	}
}

func getRequestHandler(server *HTTPServer, request *ServerHTTPRequest) (*responseHandlers, error) {
	if handlers, exists := server.uriHandlers[request.method]; exists {
		for _, handler := range handlers {
			var uriPattern = handler.uriPattern
			if isURIMatch(request.uri.Path, uriPattern) {
				return handler, nil
			}
		}
		return nil, errors.New("handler not implemented")
	} else {
		return nil, errors.New("handler not implemented")
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

func NewTLSHTTPServer(address string, config *tls.Config) (*HTTPServer, error) {
	listener, err := tls.Listen("tcp", address, config)
	if err != nil {
		return nil, err
	}
	return &HTTPServer{
		address:     address,
		listener:    listener,
		uriHandlers: make(map[string][]*responseHandlers),
	}, nil
}
