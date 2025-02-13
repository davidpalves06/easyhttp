package easyhttp

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	"net/textproto"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

// Struct that represent a HTTP Server
type HTTPServer struct {
	// Server Address
	address     string
	listener    net.Listener
	uriHandlers map[string]map[string]*responseHandler
	patterns    []string
	running     bool
	waitGroup   sync.WaitGroup
	// Server Timeout
	timeout time.Duration
}

// Function that sets server request timeout
func (s *HTTPServer) SetTimeout(timeout_ms time.Duration) {
	s.timeout = timeout_ms
}

// Function that responds to HTTP Requests
type ResponseFunction func(ServerHTTPRequest, *ServerHTTPResponse)

// Function that responds to HTTP Request Chunk
type ServerChunkFunction func([]byte, ServerHTTPRequest, *ServerHTTPResponse) bool

// Function that responds to HTTP Response Chunk on Client Requests
type ClientChunkFunction func([]byte, *ClientHTTPResponse) bool

// Additional Options for Handlers
type HandlerOptions struct {
	// Function to run on every chunk if request is chunked
	onChunk ServerChunkFunction
	// Indicates if ResponseFunction should still run after all chunks are received
	runAfterChunks bool
}

type responseHandler struct {
	uriPattern string
	handler    ResponseFunction
	options    HandlerOptions
}

// Response Function that responds to a request with the file indicated by fileName
func FileServer(fileName string) ResponseFunction {
	return func(request ServerHTTPRequest, response *ServerHTTPResponse) {
		response.statusCode = STATUS_OK
		response.SendFile(fileName)
	}
}

// Response Function that responds to a request with the file indicated by filePrefix + lastPathElement
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

// Response Function that responds to a request with a perma redirect to given redirectURI
func PermaRedirect(redirectURI string) ResponseFunction {
	return func(request ServerHTTPRequest, response *ServerHTTPResponse) {
		response.SetStatus(STATUS_MOVED_PERMANENTLY)
		response.SetHeader("Location", redirectURI)
	}
}

func (s *HTTPServer) addHandlerForMethod(handler *responseHandler, method string) {
	if !slices.Contains(s.patterns, handler.uriPattern) {
		s.patterns = append(s.patterns, handler.uriPattern)
	}

	if currentHandlers, exists := s.uriHandlers[handler.uriPattern]; exists {
		currentHandlers[method] = handler
	} else {
		currentHandlers = make(map[string]*responseHandler)
		currentHandlers[method] = handler
		s.uriHandlers[handler.uriPattern] = currentHandlers
	}
}

// Add Handler for GET method to given uri pattern
func (s *HTTPServer) HandleGET(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodGet)
}

// Add Handler for GET method to given uri pattern with additional options
func (s *HTTPServer) HandleGETWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodGet)
}

// Add Handler for POST method to given uri pattern
func (s *HTTPServer) HandlePOST(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodPost)
}

// Add Handler for POST method to given uri pattern with additional options
func (s *HTTPServer) HandlePOSTWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodPost)
}

// Add Handler for PUT method to given uri pattern
func (s *HTTPServer) HandlePUT(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodPut)
}

// Add Handler for PUT method to given uri pattern with additional options
func (s *HTTPServer) HandlePUTWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodPut)
}

// Add Handler for DELETE method to given uri pattern
func (s *HTTPServer) HandleDELETE(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodDelete)
}

// Add Handler for DELETE method to given uri pattern with additional options
func (s *HTTPServer) HandleDELETEWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodDelete)
}

// Add Handler for PATCH method to given uri pattern
func (s *HTTPServer) HandlePATCH(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandler = new(responseHandler)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodPatch)
}

// Add Handler for PATCH method to given uri pattern with additional options
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
	for _, uri := range server.patterns {
		if isURIMatch(request.uri.Path, uri) {
			matched = true
			var methodMap = server.uriHandlers[uri]
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

// Start listening to requests. This method blocks until server is closed
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

// Gracefully shutdown server waiting for open connections to finish
func (s *HTTPServer) GracefullShutdown() error {
	s.running = false
	err := s.listener.Close()
	s.waitGroup.Wait()
	return err
}

// Closes server immediatly
func (s *HTTPServer) Close() error {
	s.running = false
	err := s.listener.Close()
	return err
}

// Create a HTTP Server listening in address
func NewHTTPServer(address string) (*HTTPServer, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return &HTTPServer{
		address:     address,
		listener:    listener,
		uriHandlers: make(map[string]map[string]*responseHandler),
		patterns:    make([]string, 0, 10),
	}, nil
}

// Create a HTTPS Server listening in address
func NewTLSHTTPServer(address string, config *tls.Config) (*HTTPServer, error) {
	listener, err := tls.Listen("tcp", address, config)
	if err != nil {
		return nil, err
	}
	return &HTTPServer{
		address:     address,
		listener:    listener,
		uriHandlers: make(map[string]map[string]*responseHandler),
		patterns:    make([]string, 0, 10),
	}, nil
}
