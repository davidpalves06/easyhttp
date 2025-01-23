package gohttp

import (
	"errors"
	"net"
	"net/textproto"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

type ServerHTTPRequest struct {
	method       string
	uri          *url.URL
	version      string
	headers      Headers
	Body         []byte
	chunkChannel chan []byte
	chunked      bool
}

func (r *ServerHTTPRequest) SetHeader(key string, value string) {
	r.headers[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
}

func (r *ServerHTTPRequest) GetHeader(key string) string {
	value, found := r.headers[strings.ToLower(key)]
	if found {
		return value
	} else {
		return ""
	}
}

func (r *ServerHTTPRequest) ExistsHeader(key string) bool {
	_, found := r.headers[strings.ToLower(key)]
	return found
}

func (r *ServerHTTPRequest) Headers() Headers {
	return r.headers
}

func (r *ServerHTTPRequest) Version() string {
	return r.version
}

func (r *ServerHTTPRequest) SetVersion(version string) error {
	if slices.Contains(validVersions, version) {
		r.version = version
		return nil
	}
	return errors.New("invalid Version")
}

func (r *ServerHTTPRequest) SendChunk(chunk []byte) {
	r.chunkChannel <- chunk
}

func (r *ServerHTTPRequest) Done() {
	close(r.chunkChannel)
}

func (r *ServerHTTPRequest) Chunked() {
	r.chunked = true
}

func parseRequestLine(requestLine string, request *ServerHTTPRequest) error {
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

func parseHeaders(requestReader *textproto.Reader, request *ServerHTTPRequest) {
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

func parseRequestFromConnection(requestReader *textproto.Reader) (*ServerHTTPRequest, error) {
	var request *ServerHTTPRequest = &ServerHTTPRequest{
		headers: make(map[string]string),
	}
	requestLine, err := requestReader.ReadLine()
	if err != nil {
		return nil, err
	}
	err = parseRequestLine(requestLine, request)
	if err != nil {
		return nil, err
	}

	parseHeaders(requestReader, request)
	if request.GetHeader("Host") == "" {
		return nil, ErrParsing
	}

	return request, nil
}

func parseRequestBody(request *ServerHTTPRequest, connection net.Conn, requestReader *textproto.Reader, response *ServerHTTPResponse, onChunk ServerChunkFunction) error {
	transferEncoding := request.GetHeader("Transfer-Encoding")
	contentLengthValue := request.GetHeader("Content-Length")
	connection.SetReadDeadline(time.Now().Add(KEEP_ALIVE_TIMEOUT * time.Second))
	var err error
	if request.version == "1.1" && transferEncoding == "chunked" {
		request.Body, err = parseServerChunkedBody(requestReader, connection, request, response, onChunk)
		if err != nil {
			return err
		}
	} else if contentLengthValue != "" {

		var bodyLength, err = strconv.ParseInt(contentLengthValue, 10, 32)
		if err != nil {
			return err
		}
		if bodyLength != 0 {
			request.Body, err = parseBodyWithFullContent(bodyLength, requestReader)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
