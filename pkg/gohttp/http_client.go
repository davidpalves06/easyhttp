package gohttp

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"net/textproto"
	"strconv"
	"time"
)

var activeConnections map[string]net.Conn = make(map[string]net.Conn)

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

	request.SetHeader("Host", request.uri.Host)

	var connection net.Conn
	var err error

	connection, exists := activeConnections[request.uri.Host]
	if !exists || !checkIfConnectionIsStillOpen(connection) {
		connection, err = net.Dial("tcp", request.uri.Host)
		if err != nil {
			return nil, err
		}
	}

	requestBytes, err := request.toBytes()
	if err != nil {
		return nil, err
	}
	_, err = connection.Write(requestBytes)
	if err != nil {
		return nil, err
	}

	if request.chunked {
		request.sendChunks(connection)
	}

	var responseReader = textproto.NewReader(bufio.NewReader(connection))
	response, err := parseResponsefromConnection(responseReader)
	response.conn = connection

	transferEncoding := response.GetHeader("Transfer-Encoding")
	contentLengthValue := response.GetHeader("Content-Length")
	connection.SetReadDeadline(time.Now().Add(KEEP_ALIVE_TIMEOUT * time.Second))
	var responseBody []byte
	if response.version == "1.1" && transferEncoding == "chunked" {
		responseBody, err = parseChunkedBody(responseReader, request, response, request.onResponseChunk)
		response.body = bytes.NewBuffer(responseBody)
		if err != nil {
			return nil, err
		}
	} else if contentLengthValue != "" {
		var bodyLength, err = strconv.ParseInt(contentLengthValue, 10, 32)
		if err != nil {
			return nil, ErrParsing
		}
		if bodyLength != 0 {
			responseBody, err := parseBodyWithFullContent(bodyLength, responseReader)
			if err != nil {
				return nil, err
			}
			response.body = bytes.NewBuffer(responseBody)
		}
	} else {
		response.body = nil
	}

	if err != nil {
		return nil, err
	}
	if isClosingRequest(&request) {
		connection.Close()
		delete(activeConnections, request.uri.Host)
	} else {
		activeConnections[request.uri.Host] = connection
	}
	return response, nil
}

func checkIfConnectionIsStillOpen(connection net.Conn) bool {
	one := make([]byte, 1)
	connection.SetReadDeadline(time.Now().Add(100 * time.Microsecond))

	if _, err := connection.Read(one); err == io.EOF {
		return false
	} else {
		connection.SetReadDeadline(time.Time{})
		return true
	}
}
