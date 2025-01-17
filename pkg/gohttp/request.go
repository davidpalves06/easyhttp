package gohttp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/textproto"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

type HTTPRequest struct {
	method  string
	uri     *url.URL
	version string
	headers Headers
	Body    []byte
}

func (r *HTTPRequest) SetHeader(key string, value string) {
	r.headers[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
}

func (r *HTTPRequest) GetHeader(key string) (string, bool) {
	value, found := r.headers[strings.ToLower(key)]
	return value, found
}

func (r *HTTPRequest) Headers() Headers {
	return r.headers
}

func (r HTTPRequest) toBytes() ([]byte, error) {
	buffer := new(bytes.Buffer)
	var requestLine = fmt.Sprintf("%s %s HTTP/%s\r\n", r.method, r.uri.RequestURI(), r.version)
	buffer.WriteString(requestLine)
	r.SetHeader("Content-Length", strconv.Itoa(len(r.Body)))

	r.headers["User-Agent"] = softwareName

	for headerName, headerValue := range r.headers {
		var headerLine = fmt.Sprintf("%s: %s\r\n", headerName, headerValue)
		buffer.WriteString(headerLine)
	}

	buffer.WriteString("\r\n")

	if r.Body != nil {
		bodyLength := len(r.Body)
		if bodyLength == 0 {
			return nil, errors.New("content length is not valid")
		}

		if r.method == "GET" || r.method == "HEAD" {
			return nil, fmt.Errorf("method %s should not have a body", r.method)
		}

		buffer.Write(r.Body[:bodyLength])

	}
	return buffer.Bytes(), nil
}

func NewRequestWithBody(uri string, body []byte) (HTTPRequest, error) {
	requestURI, err := url.ParseRequestURI(uri)
	if err != nil {
		return HTTPRequest{}, errors.New("uri is not valid")
	}

	newRequest := HTTPRequest{
		headers: make(Headers),
		version: "1.0",
		Body:    body,
		uri:     requestURI,
	}

	if len(body) > 0 {
		newRequest.SetHeader("Content-Length", strconv.Itoa(len(body)))
		newRequest.SetHeader("Content-Type", "text/plain")
	}
	return newRequest, nil
}

func NewRequest(uri string) (HTTPRequest, error) {
	requestURI, err := url.ParseRequestURI(uri)
	if err != nil {
		return HTTPRequest{}, errors.New("uri is not valid")
	}
	newRequest := HTTPRequest{
		headers: make(Headers),
		version: "1.0",
		Body:    nil,
		uri:     requestURI,
	}
	return newRequest, nil
}

func newBadRequest() HTTPResponse {
	badRequestResponse := HTTPResponse{
		version:    "1.0",
		StatusCode: 400,
	}
	return badRequestResponse
}

func parseRequestLine(requestLine string, request *HTTPRequest) error {
	var requestLineSplit = strings.Split(requestLine, " ")
	if len(requestLineSplit) != 3 {
		return ErrParsing
	}
	var method string = requestLineSplit[0]
	if !slices.Contains(validMethods, method) {
		return ErrParsing
	}
	request.method = method

	var requestUri = requestLineSplit[1]
	parsedUri, err := url.ParseRequestURI(requestUri)
	if err != nil {
		return ErrParsing
	}
	request.uri = parsedUri

	var version = requestLineSplit[2]
	versionSplit := strings.Split(version, "/")
	if len(versionSplit) != 2 || versionSplit[0] != "HTTP" || !slices.Contains(validVersions, versionSplit[1]) {
		return ErrParsing
	}
	request.version = versionSplit[1]
	return nil

}

func parseHeaders(requestReader *textproto.Reader, request *HTTPRequest) {
	for {
		var line, err = requestReader.ReadLine()
		if err != nil {
			continue
		}
		if line == "" {
			break
		}
		headerSplit := strings.Split(line, ":")
		if len(headerSplit) >= 2 {
			request.SetHeader(headerSplit[0], strings.Join(headerSplit[1:], ":"))
		}
	}
}

func parseRequestFromConnection(connection net.Conn) (*HTTPRequest, error) {
	var buffer []byte = make([]byte, 2048)
	connection.SetReadDeadline(time.Now().Add(KEEP_ALIVE_TIMEOUT * time.Second))
	bytesRead, err := connection.Read(buffer)
	if err != nil || bytesRead == 0 {
		return nil, err
	}
	var request *HTTPRequest = &HTTPRequest{
		headers: make(map[string]string),
	}

	var requestReader = textproto.NewReader(bufio.NewReader(bytes.NewReader(buffer)))
	requestLine, err := requestReader.ReadLine()
	if err != nil {
		return nil, err
	}

	err = parseRequestLine(requestLine, request)
	if err != nil {
		return nil, err
	}

	parseHeaders(requestReader, request)

	contentLengthValue, hasBody := request.GetHeader("Content-Length")
	if hasBody {
		var bodyLength, err = strconv.ParseInt(contentLengthValue, 10, 32)
		if err != nil {
			return nil, ErrParsing
		}

		var bodyBuffer []byte = make([]byte, 2048)
		readBodyLength, err := requestReader.R.Read(bodyBuffer)
		if err != nil {
			return nil, err
		}

		var bodyBytes *bytes.Buffer = new(bytes.Buffer)
		if int(bodyLength) > readBodyLength {

			var readSize int = readBodyLength
			bodyBytes.Write(bodyBuffer[:readBodyLength])

			for readSize < int(bodyLength) {
				connection.SetReadDeadline(time.Now().Add(5 * time.Second))
				read, err := connection.Read(bodyBuffer)
				if (err != nil && err != io.EOF) || read == 0 {
					return nil, err
				}

				read = int(math.Min(float64(bodyLength-int64(readSize)), float64(read)))
				bodyBytes.Write(bodyBuffer[:read])
				readSize += read
			}
		} else {
			bodyBytes.Write(bodyBuffer[:bodyLength])
		}

		request.Body = bodyBytes.Bytes()

	}
	return request, nil
}

func isClosingRequest(request *HTTPRequest) bool {
	connection, exists := request.GetHeader("Connection")
	if request.version == "1.0" {
		return !(exists && connection == "keep-alive")
	} else {
		return exists && connection == "close"
	}
}
