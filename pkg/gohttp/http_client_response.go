package gohttp

import (
	"bytes"
	"errors"
	"io"
	"net/textproto"
	"slices"
	"strconv"
	"strings"
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
