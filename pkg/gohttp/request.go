package gohttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

type HTTPRequest struct {
	method  string
	uri     *url.URL
	version string
	headers Headers
	Body    io.Reader
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

	var requestLine = fmt.Sprintf("%s %s HTTP/1.0\r\n", r.method, r.uri.RequestURI())
	buffer.WriteString(requestLine)

	for headerName, headerValue := range r.headers {
		var headerLine = fmt.Sprintf("%s: %s\r\n", headerName, headerValue)
		buffer.WriteString(headerLine)
	}

	buffer.WriteString("\r\n")

	contentLengthValue, hasBody := r.GetHeader("Content-Length")

	if hasBody {
		bodyLength, err := strconv.ParseInt(contentLengthValue, 10, 32)
		if err != nil || bodyLength == 0 {
			return nil, errors.New("content length is not valid")
		}

		if r.method == "GET" || r.method == "HEAD" {
			return nil, fmt.Errorf("method %s should not have a body", r.method)
		}

		bodyBuffer := make([]byte, 1024)
		readSize, err := r.Body.Read(bodyBuffer)
		if err != nil || readSize < int(bodyLength) {
			return nil, errors.New("error with the request body")
		}

		buffer.Write(bodyBuffer[:bodyLength])
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
		version: "HTTP/1.0",
		Body:    bytes.NewReader(body),
		uri:     requestURI,
	}

	newRequest.SetHeader("Content-Length", strconv.Itoa(len(body)))
	return newRequest, nil
}

func NewRequest(uri string) (HTTPRequest, error) {
	requestURI, err := url.ParseRequestURI(uri)
	if err != nil {
		return HTTPRequest{}, errors.New("uri is not valid")
	}
	newRequest := HTTPRequest{
		headers: make(Headers),
		version: "HTTP/1.0",
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
	if len(versionSplit) != 2 || versionSplit[0] != "HTTP" || versionSplit[1] != "1.0" {
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

func parseRequestFromBytes(buffer []byte, bytesRead int) (*HTTPRequest, error) {
	var requestString = string(buffer[:bytesRead])
	requestString = strings.ReplaceAll(requestString, "\r\n", "\n")
	splitedInput := strings.Split(requestString, "\n")
	firstLineSplit := strings.Split(splitedInput[0], " ")
	var request *HTTPRequest = &HTTPRequest{
		headers: make(map[string]string),
	}
	err := parseRequestLine(firstLineSplit, request)
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
		stringBody := strings.Join(splitedInput[headerLine:], "\n")
		request.Body = bytes.NewReader([]byte(stringBody[:bodyLength]))
	}
	return request, nil
}
