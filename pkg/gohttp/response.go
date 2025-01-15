package gohttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"slices"
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
	version    string
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
	var statusLine = fmt.Sprintf("HTTP/%s %d %s\r\n", r.version, r.StatusCode, reasonPhrase)
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

		bodyBuffer := make([]byte, 2048)
		var readSize int64

		for readSize < bodyLength {
			read, err := r.Body.Read(bodyBuffer)
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

func newHTTPResponse(responseWriter HTTPResponseWriter) HTTPResponse {
	var response = HTTPResponse{
		headers: make(Headers),
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

	if responseWriter.buffer != nil && responseWriter.buffer.Len() > 0 {
		response.SetHeader("Content-Type", "text/plain")
		response.SetHeader("Content-Length", strconv.Itoa(responseWriter.buffer.Len()))
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
	if len(versionSplit) != 2 || versionSplit[0] != "HTTP" || !slices.Contains(validVersions, versionSplit[1]) {
		return errors.New("invalid HTTP Version")
	}

	response.version = versionSplit[1]

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

func parseResponsefromConnection(connection net.Conn) (*HTTPResponse, error) {
	var response = &HTTPResponse{
		headers: make(Headers),
	}
	var buffer []byte = make([]byte, 2048)
	bytesRead, err := connection.Read(buffer)
	if err != nil {
		log.Println("Failed to read response", err)
		return nil, err
	}

	var responseString = string(buffer[:bytesRead])
	splitedInput := strings.Split(responseString, "\r\n")
	firstLineSplit := strings.Split(splitedInput[0], " ")
	err = parseResponseStatusLine(firstLineSplit, response)
	if err != nil {
		return nil, err
	}

	headerLine := parseResponseHeaders(splitedInput, response)

	contentLengthValue, hasBody := response.GetHeader("Content-Length")

	if hasBody {
		bodyLength, err := strconv.ParseInt(contentLengthValue, 10, 32)
		if err != nil || bodyLength == 0 {
			return nil, errors.New("content length is not valid")
		}
		stringBody := strings.Join(splitedInput[headerLine:], "\r\n")

		var bodyBuffer *bytes.Buffer = new(bytes.Buffer)

		if int(bodyLength) > len(stringBody) {
			var readSize int = len(stringBody)
			bodyBuffer.Write([]byte(stringBody[:readSize]))

			for readSize < int(bodyLength) {
				read, err := connection.Read(buffer)
				if err != nil && err != io.EOF {
					return nil, errors.New("error with the request body")
				}

				read = int(math.Min(float64(bodyLength-int64(readSize)), float64(read)))
				bodyBuffer.Write(buffer[:read])
				readSize += read
			}
		} else {
			bodyBuffer.Write([]byte(stringBody[:bodyLength]))
		}

		response.Body = bodyBuffer
	} else {
		response.Body = nil
	}

	return response, nil
}
