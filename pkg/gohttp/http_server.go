package gohttp

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
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
	uriHandlers map[string][]*responseHandlers
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

func PermaRedirect(redirectURI string) ResponseFunction {
	return func(request ServerHTTPRequest, response *ServerHTTPResponse) {
		response.SetStatus(STATUS_MOVED_PERMANENTLY)
		response.SetHeader("Location", redirectURI)
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

func (s *HTTPServer) HandlePUT(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandlers = new(responseHandlers)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodPut)
}

func (s *HTTPServer) HandlePUTWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandlers = new(responseHandlers)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodPut)
}

func (s *HTTPServer) HandleDELETE(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandlers = new(responseHandlers)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodDelete)
}

func (s *HTTPServer) HandleDELETEWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandlers = new(responseHandlers)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodDelete)
}

func (s *HTTPServer) HandlePATCH(uriPattern string, handlerFunction ResponseFunction) {
	var handler *responseHandlers = new(responseHandlers)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options.onChunk = nil

	s.addHandlerForMethod(handler, MethodPatch)
}

func (s *HTTPServer) HandlePATCHWithOptions(uriPattern string, handlerFunction ResponseFunction, options HandlerOptions) {
	var handler *responseHandlers = new(responseHandlers)
	handler.uriPattern = uriPattern
	handler.handler = handlerFunction
	handler.options = options

	s.addHandlerForMethod(handler, MethodPatch)
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
			sendErrorResponse(err, connection)
			return
		}
		response := newHTTPResponse(request, connection)

		handler, err := getRequestHandler(server, request)
		if err != nil {
			response.statusCode = STATUS_METHOD_NOT_ALLOWED
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

func executeRequest(server *HTTPServer, handler *responseHandlers, request *ServerHTTPRequest, response *ServerHTTPResponse, connection net.Conn) error {
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
		} else if err == ErrInvalidMethod {
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
	//TODO:BETTER SHUTDOWN LOGIC
	// s.waitGroup.Wait()
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
