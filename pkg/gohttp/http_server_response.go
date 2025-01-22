package gohttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"strconv"
	"strings"
	"time"
)

type ServerHTTPResponse struct {
	headers     Headers
	statusCode  int
	version     string
	body        *bytes.Buffer
	chunkWriter io.Writer
	chunked     bool
	//TODO: CHECK HOW TO DO CHUNKED HEAD RESPONSE
	method string
}

func (r *ServerHTTPResponse) Write(p []byte) (n int, err error) {
	return r.body.Write(p)
}

func (r *ServerHTTPResponse) SendChunk() (int, error) {
	if r.method == MethodHead {
		return 0, errors.New("head message cannot be chunked")
	}
	if !r.chunked {
		r.chunked = true
		responseBytes, err := r.toBytes()
		if err != nil {
			return 0, err
		}
		r.chunkWriter.Write(responseBytes)
	}

	buffer := new(bytes.Buffer)
	var chunkLength = r.body.Len()
	if chunkLength <= 0 {
		return 0, errors.New("chunk size cannot be 0")
	}
	chunkLengthLine := fmt.Sprintf("%x \r\n", chunkLength)
	buffer.WriteString(chunkLengthLine)
	buffer.Write(r.body.Bytes())

	buffer.WriteString("\r\n")

	r.chunkWriter.Write(buffer.Bytes())

	r.body.Reset()
	return chunkLength, nil
}

func (r *ServerHTTPResponse) HasBody() bool {
	return r.body != nil && r.body.Len() != 0
}

func (r *ServerHTTPResponse) Read(buffer []byte) (int, error) {
	return r.body.Read(buffer)
}

func (r *ServerHTTPResponse) SetHeader(headerName string, headerValue string) {
	r.headers[strings.ToLower(strings.TrimSpace(headerName))] = strings.TrimSpace(headerValue)
}

func (r *ServerHTTPResponse) SetStatus(status int) {
	r.statusCode = status
}

func (r *ServerHTTPResponse) GetHeader(key string) string {
	value, found := r.headers[strings.ToLower(key)]
	if found {
		return value
	} else {
		return ""
	}
}

func (r *ServerHTTPResponse) ExistsHeader(key string) bool {
	_, found := r.headers[strings.ToLower(key)]
	return found
}

func (r *ServerHTTPResponse) Headers() Headers {
	return r.headers
}

func (r *ServerHTTPResponse) toBytes() ([]byte, error) {
	buffer := new(bytes.Buffer)
	var reasonPhrase = reasons[r.statusCode]
	var statusLine = fmt.Sprintf("HTTP/%s %d %s\r\n", r.version, r.statusCode, reasonPhrase)
	buffer.WriteString(statusLine)

	addEssentialHTTPHeaders(r)

	if r.chunked {
		r.SetHeader("Transfer-Encoding", "chunked")
	} else if r.body != nil && r.body.Len() > 0 {
		r.SetHeader("Content-Length", strconv.Itoa(r.body.Len()))
	}

	for headerName, headerValue := range r.headers {
		var headerLine = fmt.Sprintf("%s: %s\r\n", headerName, headerValue)
		buffer.WriteString(headerLine)
	}

	buffer.WriteString("\r\n")

	contentLengthValue := r.GetHeader("Content-Length")
	if contentLengthValue != "" && !r.chunked {
		bodyLength, err := strconv.ParseInt(contentLengthValue, 10, 32)
		if err != nil || bodyLength == 0 {
			return nil, errors.New("content length not valid")
		}

		bodyBuffer := make([]byte, 2048)
		var readSize int64

		for readSize < bodyLength {
			read, err := r.body.Read(bodyBuffer)
			if err != nil && err != io.EOF {
				return nil, errors.New("error with the request body")
			}

			read = int(math.Min(float64(bodyLength-readSize), float64(read)))
			buffer.Write(bodyBuffer[:read])
			readSize += int64(read)
		}
	}

	return buffer.Bytes(), nil
}

func newHTTPResponse(request *ServerHTTPRequest, connection net.Conn) *ServerHTTPResponse {
	response := &ServerHTTPResponse{
		headers:     make(map[string]string),
		statusCode:  STATUS_OK,
		body:        new(bytes.Buffer),
		chunkWriter: connection,
		version:     request.version,
		method:      request.method,
	}
	return response
}

func newBadRequestResponse() ServerHTTPResponse {
	badRequestResponse := ServerHTTPResponse{
		version:    "1.0",
		statusCode: 400,
		headers:    make(Headers),
	}
	return badRequestResponse
}

func addEssentialHTTPHeaders(response *ServerHTTPResponse) {

	response.SetHeader("Date", time.Now().UTC().Format(time.RFC1123))
	response.SetHeader("Server", softwareName)

	if response.statusCode == STATUS_UNAUTHORIZED {
		if _, exists := response.headers["WWW-Authenticate"]; !exists {
			log.Printf("Warning : Status 401 has no WWW-Authenticate header\n")
		}
	}
}
