package gohttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type HTTPResponseWriter struct {
	headers    Headers
	statusCode int
	buffer     *bytes.Buffer
}

func (r *HTTPResponseWriter) Write(p []byte) (n int, err error) {
	return r.buffer.Write(p)
}

func (r *HTTPResponseWriter) SetHeader(headerName string, headerValue string) {
	r.headers[strings.ToLower(headerName)] = headerValue
}

func (r *HTTPResponseWriter) SetStatus(status int) {
	r.statusCode = status
}

var reasons = map[int]string{
	200: "OK",
}

type HTTPResponse struct {
	Headers    Headers
	StatusCode int
	Body       io.Reader
}

func (r HTTPResponse) toBytes() []byte {
	buffer := new(bytes.Buffer)
	var reasonPhrase = reasons[r.StatusCode]
	var statusLine = fmt.Sprintf("HTTP/1.0 %d %s\r\n", r.StatusCode, reasonPhrase)
	buffer.WriteString(statusLine)

	for headerName, headerValue := range r.Headers {
		var headerLine = fmt.Sprintf("%s: %s\r\n", headerName, headerValue)
		buffer.WriteString(headerLine)
	}

	buffer.WriteString("\r\n")

	bodyBuffer := make([]byte, 1024)
	bodySize, err := r.Body.Read(bodyBuffer)
	if err != nil {
		fmt.Printf("ERROR :%s\n", err.Error())
	}

	buffer.Write(bodyBuffer[:bodySize])
	return buffer.Bytes()
}

func newHTTPResponse(responseWriter HTTPResponseWriter) HTTPResponse {
	var response = HTTPResponse{
		Headers: make(Headers),
	}
	for headerName, headerValue := range responseWriter.headers {
		response.Headers[headerName] = headerValue
	}
	response.StatusCode = responseWriter.statusCode
	response.Headers["Content-Length"] = strconv.Itoa(responseWriter.buffer.Len())
	response.Body = responseWriter.buffer
	return response
}

func parseResponseStatusLine(firstLineSplit []string, response *HTTPResponse) error {
	if len(firstLineSplit) != 3 {
		return errors.New("incomplete Status Line")
	}
	var version string = firstLineSplit[0]
	versionSplit := strings.Split(version, "/")
	if len(versionSplit) != 2 || versionSplit[0] != "HTTP" || versionSplit[1] != "1.0" {
		return errors.New("invalid HTTP Version")
	}

	var statusCode = firstLineSplit[1]
	parsedStatus, err := strconv.ParseInt(statusCode, 10, 16)
	if err != nil || parsedStatus < 100 || parsedStatus >= 600 {
		return errors.New("invalid StatusCode")
	}
	response.StatusCode = int(parsedStatus)

	return nil
}

func parseResponseHeaders(splitedInput []string, response *HTTPResponse) uint8 {
	var headerLine uint8 = 1
	for _, line := range splitedInput[1:] {
		if line == "" {
			headerLine += 1
			break
		}
		headerSplit := strings.Split(line, ":")
		response.Headers[headerSplit[0]] = strings.TrimSpace(strings.Join(headerSplit[1:], ":"))
		headerLine += 1
	}
	return headerLine
}

func parseResponsefromBytes(responseBytes []byte) HTTPResponse {
	var response = HTTPResponse{
		Headers: make(Headers),
	}
	bytesReader := bytes.NewReader(responseBytes)
	buffer := make([]byte, 1024)
	bytesRead, err := bytesReader.Read(buffer)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

	var responseString = string(buffer[:bytesRead])
	responseString = strings.ReplaceAll(responseString, "\r\n", "\n")
	splitedInput := strings.Split(responseString, "\n")
	firstLineSplit := strings.Split(splitedInput[0], " ")
	err = parseResponseStatusLine(firstLineSplit, &response)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

	headerLine := parseResponseHeaders(splitedInput, &response)
	response.Body = bytes.NewReader([]byte(strings.Join(splitedInput[headerLine:], "\n")))
	return response
}
