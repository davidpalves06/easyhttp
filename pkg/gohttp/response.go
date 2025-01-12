package gohttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"
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

type HTTPResponse struct {
	headers    Headers
	StatusCode int
	Body       io.Reader
}

func (r *HTTPResponse) SetHeader(key string, value string) {
	r.headers[strings.ToLower(key)] = value
}

func (r *HTTPResponse) GetHeader(key string) (string, bool) {
	value, found := r.headers[strings.ToLower(key)]
	return value, found
}

func (r *HTTPResponse) Headers() Headers {
	return r.headers
}

func (r HTTPResponse) toBytes() ([]byte, error) {
	buffer := new(bytes.Buffer)
	var reasonPhrase = reasons[r.StatusCode]
	var statusLine = fmt.Sprintf("HTTP/1.0 %d %s\r\n", r.StatusCode, reasonPhrase)
	buffer.WriteString(statusLine)

	for headerName, headerValue := range r.headers {
		var headerLine = fmt.Sprintf("%s: %s\r\n", headerName, headerValue)
		buffer.WriteString(headerLine)
	}

	buffer.WriteString("\r\n")

	contentLengthValue, hasBody := r.GetHeader("Content-Length")

	if hasBody {
		bodyLength, err := strconv.ParseInt(contentLengthValue, 10, 32)
		if err != nil || bodyLength == 0 {
			return nil, errors.New("content length not valid")
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

func newHTTPResponse(responseWriter HTTPResponseWriter) HTTPResponse {
	var response = HTTPResponse{
		headers: make(Headers),
	}

	if responseWriter.buffer.Len() > 0 {
		response.SetHeader("Content-Type", "text/plain")
		response.SetHeader("Content-Length", strconv.Itoa(responseWriter.buffer.Len()))
	}

	response.SetHeader("Date", time.Now().UTC().Format(time.RFC1123))
	response.SetHeader("Server", softwareName)

	if responseWriter.statusCode == STATUS_UNAUTHORIZED {
		if _, exists := responseWriter.headers["WWW-Authenticate"]; !exists {
			log.Printf("Warning : Status 401 has no WWW-Authenticate header\n")
		}
	}

	for headerName, headerValue := range responseWriter.headers {
		response.SetHeader(headerName, headerValue)
	}

	response.StatusCode = responseWriter.statusCode
	response.Body = responseWriter.buffer
	return response
}

func parseResponseStatusLine(firstLineSplit []string, response *HTTPResponse) error {
	if len(firstLineSplit) < 3 {
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
		response.SetHeader(headerSplit[0], strings.TrimSpace(strings.Join(headerSplit[1:], ":")))
		headerLine += 1
	}
	return headerLine
}

func parseResponsefromBytes(responseBytes []byte) (*HTTPResponse, error) {
	var response = &HTTPResponse{
		headers: make(Headers),
	}
	bytesReader := bytes.NewReader(responseBytes)
	buffer := make([]byte, 1024)
	bytesRead, err := bytesReader.Read(buffer)
	if err != nil {
		return nil, err
	}

	var responseString = string(buffer[:bytesRead])
	responseString = strings.ReplaceAll(responseString, "\r\n", "\n")
	splitedInput := strings.Split(responseString, "\n")
	firstLineSplit := strings.Split(splitedInput[0], " ")
	err = parseResponseStatusLine(firstLineSplit, response)
	if err != nil {
		return nil, err
	}

	headerLine := parseResponseHeaders(splitedInput, response)

	contentLengthValue, hasBody := response.GetHeader("Content-Length")
	bodyLength, err := strconv.ParseInt(contentLengthValue, 10, 32)
	if err != nil {
		return nil, errors.New("content length is not parsable")
	}

	if hasBody && bodyLength != 0 {
		stringBody := strings.Join(splitedInput[headerLine:], "\n")
		response.Body = bytes.NewReader([]byte(stringBody[:bodyLength]))
	} else {
		response.Body = nil
	}

	return response, nil
}
