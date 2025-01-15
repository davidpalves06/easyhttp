package gohttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
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
	r.headers[strings.ToLower(key)] = value
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

func parseRequestLine(firstLineSplit []string, request *HTTPRequest) error {
	if len(firstLineSplit) != 3 {
		return errors.New("incomplete Request Targets")
	}
	var method string = firstLineSplit[0]
	if !slices.Contains(validMethods, method) {
		return errors.New("invalid HTTP Method")
	}
	request.method = method

	var requestUri = firstLineSplit[1]
	parsedUri, err := url.ParseRequestURI(requestUri)
	if err != nil {
		return errors.New("invalid Uri")
	}
	request.uri = parsedUri

	var version = firstLineSplit[2]
	versionSplit := strings.Split(version, "/")
	if len(versionSplit) != 2 || versionSplit[0] != "HTTP" || !slices.Contains(validVersions, versionSplit[1]) {
		return errors.New("invalid HTTP Version")
	}
	request.version = versionSplit[1]
	return nil

}

func parseHeaders(splitedInput []string, request *HTTPRequest) uint8 {
	var headerLine uint8 = 1
	for _, line := range splitedInput[1:] {
		if line == "" {
			headerLine += 1
			break
		}
		headerSplit := strings.Split(line, ":")
		request.SetHeader(headerSplit[0], strings.TrimSpace(strings.Join(headerSplit[1:], ":")))
		headerLine += 1
	}
	return headerLine
}

func parseRequestFromConnection(connection net.Conn) (*HTTPRequest, error) {
	var buffer []byte = make([]byte, 2048)
	connection.SetReadDeadline(time.Now().Add(5 * time.Second))
	bytesRead, err := connection.Read(buffer)
	if err != nil || bytesRead == 0 {
		return nil, err
	}
	var request *HTTPRequest = &HTTPRequest{
		headers: make(map[string]string),
	}

	var requestString = string(buffer[:bytesRead])
	splitedInput := strings.Split(requestString, "\r\n")

	firstLineSplit := strings.Split(splitedInput[0], " ")

	err = parseRequestLine(firstLineSplit, request)
	if err != nil {
		return nil, err
	}

	headerLine := parseHeaders(splitedInput, request)

	contentLengthValue, hasBody := request.GetHeader("Content-Length")
	if hasBody {
		var bodyLength, err = strconv.ParseInt(contentLengthValue, 10, 32)
		if err != nil {
			return nil, errors.New("content length is not parsable")
		}

		stringBody := strings.Join(splitedInput[headerLine:], "\r\n")

		var bodyBuffer *bytes.Buffer = new(bytes.Buffer)
		if int(bodyLength) > len(stringBody) {

			var readSize int = len(stringBody)
			bodyBuffer.Write([]byte(stringBody))
			for readSize < int(bodyLength) {
				connection.SetReadDeadline(time.Now().Add(5 * time.Second))
				read, err := connection.Read(buffer)
				if (err != nil && err != io.EOF) || read == 0 {
					return nil, errors.New("error with the request body")
				}

				read = int(math.Min(float64(bodyLength-int64(readSize)), float64(read)))
				bodyBuffer.Write(buffer[:read])
				readSize += read
			}
		} else {
			bodyBuffer.Write([]byte(stringBody[:bodyLength]))
		}

		request.Body = bodyBuffer.Bytes()

	}
	return request, nil
}
