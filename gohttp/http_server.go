package gohttp

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	"net/textproto"
	"runtime"
	"strings"
	"sync"
	"time"
)

type HTTPServer struct {
	address     string
	listener    net.Listener
	uriHandlers map[string]map[string]*responseHandler
	running     bool
	waitGroup   sync.WaitGroup
	timeout     time.Duration
}

func (s *HTTPServer) SetTimeout(timeout_ms time.Duration) {
	s.timeout = timeout_ms
}

type ResponseFunction func(ServerHTTPRequest, *ServerHTTPResponse)
type ServerChunkFunction func([]byte, ServerHTTPRequest, *ServerHTTPResponse) bool
type ClientChunkFunction func([]byte, *ClientHTTPResponse) bool

type HandlerOptions struct {
	onChunk        ServerChunkFunction
	runAfterChunks bool
}

type responseHandler struct {
	uriPattern string
	handler    ResponseFunction
	options    HandlerOptions
}

func FileServer(fileName string) ResponseFunction {
	return func(request ServerHTTPRequest, response *ServerHTTPResponse) {
		response.statusCode = STATUS_OK
		response.SendFile(fileName)
	}
}

func FileServerFromPath(filePrefix string) ResponseFunction {
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

func PermaRedirect(redirectURI string) ResponseFunction {
	return func(request ServerHTTPRequest, response *ServerHTTPResponse) {
		response.SetStatus(STATUS_MOVED_PERMANENTLY)
		response.SetHeader("Location", redirectURI)
	}
}

func (s *HTTPServer) addHandlerForMethod(handler *responseHandler, method string) {

	if currentHandlers, exists := s.uriHandlers[handler.uriPattern]; exists {
		currentHandlers[method] = handler
	} else {
		currentHandlers = make(map[string]*responseHandler)
		currentHandlers[method] = handler
		s.uriHandlers[handler.uriPattern] = currentHandlers
	}
}

func (s *HTTPServer) HandleGET(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodGet)
}

func (s *HTTPServer) HandleGETWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodGet)
}

func (s *HTTPServer) HandlePOST(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodPost)
}

func (s *HTTPServer) HandlePOSTWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodPost)
}

func (s *HTTPServer) HandlePUT(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodPut)
}

func (s *HTTPServer) HandlePUTWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodPut)
}

func (s *HTTPServer) HandleDELETE(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodDelete)
}

func (s *HTTPServer) HandleDELETEWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodDelete)
}

func (s *HTTPServer) HandlePATCH(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodPatch)
}

func (s *HTTPServer) HandlePATCHWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodPatch)
}

func handleConnection(connection net.Conn, server *HTTPServer) {
	defer connection.Close()
	defer server.waitGroup.Done()
	var keepAlive = true
	for server.running && keepAlive {
		var requestReader = textproto.NewReader(bufio.NewReader(connection))
		connection.SetReadDeadline(time.Now().Add(KEEP_ALIVE_TIMEOUT * time.Second))
		request, err := parseRequestFromConnection(requestReader)
		if err != nil {
			sendErrorResponse(err, connection)
			return
		}
		response := newHTTPResponse(request, connection)

		handler, err := getRequestHandler(server, request)
		if err != nil {
			if err == ErrNotFound {
				response.statusCode = STATUS_NOT_FOUND
			}
			if err == ErrMethodNotAllowed {
				response.statusCode = STATUS_METHOD_NOT_ALLOWED
				methods := getAllowedMethods(server, request)
				for _, method := range methods {
					response.AddHeader("Allow", method)
				}
			}
		} else {
			err := parseRequestBody(request, connection, requestReader, response, handler.options.onChunk)
			if err != nil {
				sendErrorResponse(err, connection)
				return
			}

			err = executeRequest(server, handler, request, response, connection)
			if err != nil {
				sendErrorResponse(err, connection)
				return
			}
		}
		if request.method == MethodHead {
			response.body = nil
		}

		if !response.chunked {
			responseBytes, err := response.toBytes()
			if err != nil {
				sendErrorResponse(ErrInternalError, connection)
				return
			}
			connection.Write(responseBytes)
		} else {
			connection.Write([]byte("0 \r\n\r\n"))
		}
		keepAlive = !isClosingRequest(request)
	}
}

func executeRequest(server *HTTPServer, handler *responseHandler, request *ServerHTTPRequest, response *ServerHTTPResponse, connection net.Conn) error {
	var executionContext context.Context
	var executionChannel chan error = make(chan error, 1)
	if server.timeout > 0 {
		var cancel context.CancelFunc
		executionContext, cancel = context.WithTimeout(context.Background(), server.timeout)
		defer cancel()
	} else {
		executionContext = context.Background()
	}

	if handler.options.onChunk == nil || handler.options.runAfterChunks {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					executionChannel <- ErrInternalError
					runtime.Goexit()
				}
			}()
			handler.handler(*request, response)
			executionChannel <- nil
		}()
		select {
		case <-executionContext.Done():
			sendErrorResponse(ErrRequestTimeout, connection)
			return ErrRequestTimeout
		case executionError := <-executionChannel:
			if executionError != nil {
				sendErrorResponse(executionError, connection)
				return executionError
			}
		}
	}
	close(executionChannel)
	return nil
}

func sendErrorResponse(err error, connection net.Conn) {
	if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		var errorResponse ServerHTTPResponse
		if err == ErrInvalidLength {
			errorResponse = newInvalidLengthResponse()
		} else if err == ErrInvalidMethod || err == ErrMethodNotAllowed {
			errorResponse = newInvalidMethodResponse()
		} else if err == ErrVersionNotSupported {
			errorResponse = newUnsupportedVersionResponse()
		} else if err == ErrBadRequest {
			errorResponse = newBadRequestResponse()
		} else if err == ErrRequestTimeout {
			errorResponse = newRequestTimeoutErrorResponse()
		} else {
			errorResponse = newInternalErrorResponse()
		}
		responseBytes, _ := errorResponse.toBytes()
		connection.Write(responseBytes)
	}
}

func getRequestHandler(server *HTTPServer, request *ServerHTTPRequest) (*responseHandler, error) {
	var method = request.method
	if method == MethodHead {
		method = MethodGet
	}
	var matched = false
	for uri, methodMap := range server.uriHandlers {
		if isURIMatch(request.uri.Path, uri) {
			matched = true
			if handler, ok := methodMap[method]; ok {
				return handler, nil
			}
		}
	}
	if matched {
		return nil, ErrMethodNotAllowed
	} else {
		return nil, ErrNotFound
	}
}

func getAllowedMethods(server *HTTPServer, request *ServerHTTPRequest) []string {

	var methods = make([]string, 0, 5)
	for uri, methodMap := range server.uriHandlers {
		if isURIMatch(request.uri.Path, uri) {
			for method := range methodMap {
				methods = append(methods, method)
			}
			return methods
		}
	}
	return methods
}

func (s *HTTPServer) acceptConnection() (net.Conn, error) {
	return s.listener.Accept()
}

func (s *HTTPServer) Run() {
	s.running = true
	for s.running {
		connection, err := s.acceptConnection()
		if err != nil {
			break
		}
		s.waitGroup.Add(1)
		go handleConnection(connection, s)
	}
}

func (s *HTTPServer) GracefullShutdown() error {
	s.running = false
	err := s.listener.Close()
	s.waitGroup.Wait()
	return err
}

func (s *HTTPServer) Close() error {
	s.running = false
	err := s.listener.Close()
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
		uriHandlers: make(map[string]map[string]*responseHandler),
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
		uriHandlers: make(map[string]map[string]*responseHandler),
	}, nil
}
