package gohttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"slices"
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

func (r HTTPRequest) toBytes() []byte {
	buffer := new(bytes.Buffer)

	var requestLine = fmt.Sprintf("%s %s HTTP/1.0\r\n", r.method, r.uri.RequestURI())
	buffer.WriteString(requestLine)

	for headerName, headerValue := range r.headers {
		var headerLine = fmt.Sprintf("%s: %s\r\n", headerName, headerValue)
		buffer.WriteString(headerLine)
	}

	buffer.WriteString("\r\n")

	if r.method != "GET" {

		bodyBuffer := make([]byte, 1024)
		bodySize, err := r.Body.Read(bodyBuffer)
		if err != nil {
			fmt.Printf("ERROR :%s\n", err.Error())
		}

		buffer.Write(bodyBuffer[:bodySize])
	}
	return buffer.Bytes()
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
		request.headers[headerSplit[0]] = strings.TrimSpace(strings.Join(headerSplit[1:], ":"))
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
	request.Body = bytes.NewReader([]byte(strings.Join(splitedInput[headerLine:], "\n")))
	return request, nil
}
