package gohttp

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"net/textproto"
	"slices"
	"strconv"
	"strings"
	"time"
)

type ClientHTTPResponse struct {
	headers    Headers
	StatusCode int
	body       *bytes.Buffer
	version    string
}

func (r *ClientHTTPResponse) HasBody() bool {
	return r.body != nil && r.body.Len() > 0
}

func (r *ClientHTTPResponse) GetBody() io.Reader {
	return r.body
}

func (r *ClientHTTPResponse) Version() string {
	return r.version
}

func (r *ClientHTTPResponse) Read(buffer []byte) (int, error) {
	return r.body.Read(buffer)
}

func (r *ClientHTTPResponse) SetHeader(headerName string, headerValue string) {
	r.headers[strings.ToLower(strings.TrimSpace(headerName))] = strings.TrimSpace(headerValue)
}

func (r *ClientHTTPResponse) GetHeader(key string) string {
	value, found := r.headers[strings.ToLower(key)]
	if found {
		return value
	} else {
		return ""
	}
}

func (r *ClientHTTPResponse) ExistsHeader(key string) bool {
	_, found := r.headers[strings.ToLower(key)]
	return found
}

func (r *ClientHTTPResponse) Headers() Headers {
	return r.headers
}

func parseResponse(connection net.Conn, request ClientHTTPRequest) (*ClientHTTPResponse, error) {
	var responseReader = textproto.NewReader(bufio.NewReader(connection))
	response, err := parseResponsefromConnection(responseReader)
	if err != nil {
		return nil, err
	}

	err = parseResponseBody(response, connection, responseReader, request.onResponseChunk)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func parseResponseStatusLine(statusLine string, response *ClientHTTPResponse) error {
	var firstLineSplit = strings.Split(statusLine, " ")
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

func parseResponseHeaders(responseReader *textproto.Reader, response *ClientHTTPResponse) {
	for {
		var line, err = responseReader.ReadLine()
		if err != nil {
			continue
		}
		if line == "" {
			break
		}
		headerSplit := strings.Split(line, ":")
		response.SetHeader(headerSplit[0], strings.Join(headerSplit[1:], ":"))
	}
}

func parseResponsefromConnection(responseReader *textproto.Reader) (*ClientHTTPResponse, error) {
	var response = &ClientHTTPResponse{
		headers: make(Headers),
		body:    nil,
	}

	statusLine, err := responseReader.ReadLine()
	if err != nil {
		return nil, err
	}

	err = parseResponseStatusLine(statusLine, response)
	if err != nil {
		return nil, err
	}

	parseResponseHeaders(responseReader, response)

	return response, nil
}

func parseResponseBody(response *ClientHTTPResponse, connection net.Conn, responseReader *textproto.Reader, onResponseChunk ClientChunkFunction) error {
	transferEncoding := response.GetHeader("Transfer-Encoding")
	contentLengthValue := response.GetHeader("Content-Length")
	var err error
	connection.SetReadDeadline(time.Now().Add(KEEP_ALIVE_TIMEOUT * time.Second))
	if response.version == "1.1" && transferEncoding == "chunked" {
		response.body, err = parseClientChunkedBody(responseReader, connection, response, onResponseChunk)
		if err != nil {
			return err
		}
	} else if contentLengthValue != "" {
		var bodyLength, err = strconv.ParseInt(contentLengthValue, 10, 32)
		if err != nil {
			return err
		}
		if bodyLength != 0 {
			responseBody, err := parseBodyWithFullContent(bodyLength, responseReader)
			if err != nil {
				return err
			}
			response.body = bytes.NewBuffer(responseBody)
		}
	} else {
		response.body = nil
	}
	return nil
}
