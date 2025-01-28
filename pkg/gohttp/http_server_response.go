package gohttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
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
	method      string
	cookies     []*Cookie
}

func (r *ServerHTTPResponse) Write(p []byte) (n int, err error) {
	return r.body.Write(p)
}

func (r *ServerHTTPResponse) SendFile(fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	r.body.Write(fileBytes)
	return nil
}

func (r *ServerHTTPResponse) SetCookie(cookie *Cookie) {
	r.cookies = append(r.cookies, cookie)
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

func (r *ServerHTTPResponse) SetHeader(key string, value string) {
	r.headers[strings.ToLower(strings.TrimSpace(key))] = []string{strings.TrimSpace(value)}
}

func (r *ServerHTTPResponse) SetStatus(status int) {
	r.statusCode = status
}

func (r *ServerHTTPResponse) GetHeader(key string) []string {
	value, found := r.headers[strings.ToLower(key)]
	if found {
		return value
	} else {
		return nil
	}
}

func (r *ServerHTTPResponse) AddHeader(key string, value string) {
	headers, exists := r.headers[strings.ToLower(strings.TrimSpace(key))]
	if !exists {
		headers = []string{}
	}
	headers = append(headers, value)
	r.headers[strings.ToLower(strings.TrimSpace(key))] = headers
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
		builder := new(strings.Builder)
		builder.WriteString(headerName)
		builder.WriteString(": ")
		for i, value := range headerValue {
			builder.WriteString(value)
			if i < len(headerValue)-1 {
				builder.WriteString(", ")
			}
		}
		builder.WriteString("\r\n")
		buffer.WriteString(builder.String())
	}

	for _, cookie := range r.cookies {
		cookieBuilder := new(strings.Builder)
		cookieBuilder.WriteString("Set-Cookie: ")
		cookieBuilder.WriteString(cookie.String())
		cookieBuilder.WriteString("\r\n")
		buffer.WriteString(cookieBuilder.String())
	}

	buffer.WriteString("\r\n")

	contentLengthHeader := r.GetHeader("Content-Length")

	if contentLengthHeader != nil && !r.chunked {
		contentLengthValue := contentLengthHeader[len(contentLengthHeader)-1]
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
		headers:     make(map[string][]string),
		statusCode:  STATUS_OK,
		body:        new(bytes.Buffer),
		chunkWriter: connection,
		version:     request.version,
		method:      request.method,
		cookies:     make([]*Cookie, 0, 5),
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

func newInternalErrorResponse() ServerHTTPResponse {
	badRequestResponse := ServerHTTPResponse{
		version:    "1.0",
		statusCode: 500,
		headers:    make(Headers),
	}
	return badRequestResponse
}

func newInvalidLengthResponse() ServerHTTPResponse {
	badRequestResponse := ServerHTTPResponse{
		version:    "1.0",
		statusCode: STATUS_LENGTH_REQUIRED,
		headers:    make(Headers),
	}
	return badRequestResponse
}

func newInvalidMethodResponse() ServerHTTPResponse {
	badRequestResponse := ServerHTTPResponse{
		version:    "1.0",
		statusCode: STATUS_METHOD_NOT_ALLOWED,
		headers:    make(Headers),
	}
	return badRequestResponse
}

func newUnsupportedVersionResponse() ServerHTTPResponse {
	badRequestResponse := ServerHTTPResponse{
		version:    "1.0",
		statusCode: STATUS_HTTP_VERSION_NOT_SUPPORTED,
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
