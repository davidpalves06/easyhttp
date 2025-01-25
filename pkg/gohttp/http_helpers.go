package gohttp

import (
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

type httpRequest interface {
	SetHeader(key string, value string)
	GetHeader(key string) []string
	Version() string
	SetVersion(version string) error
	HasHeaderValue(key string, value string) bool
}

type Headers map[string][]string

const softwareName = "GoHTTP 1.0"
const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodPatch   = "PATCH"
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
)

var validMethods = []string{"GET", "HEAD", "POST"}
var validVersions = []string{"1.0", "1.1", "2.0"}

const (
	STATUS_OK                  = 200
	STATUS_CREATED             = 201
	STATUS_ACCEPTED            = 202
	STATUS_NO_CONTENT          = 204
	STATUS_MULTIPLE_CHOICES    = 300
	STATUS_MOVED_PERMANENTLY   = 301
	STATUS_MOVED_TEMPORARILY   = 302
	STATUS_NOT_MODIFIED        = 304
	STATUS_BAD_REQUEST         = 400
	STATUS_UNAUTHORIZED        = 401
	STATUS_FORBIDDEN           = 403
	STATUS_NOT_FOUND           = 404
	STATUS_INTERNAL_ERROR      = 500
	STATUS_NOT_IMPLEMENTED     = 501
	STATUS_BAD_GATEWAY         = 502
	STATUS_SERVICE_UNAVAILABLE = 503
)

var reasons = map[int]string{
	STATUS_OK:                  "OK",
	STATUS_CREATED:             "Created",
	STATUS_ACCEPTED:            "Accepted",
	STATUS_NO_CONTENT:          "No content",
	STATUS_MULTIPLE_CHOICES:    "Multiple Choices",
	STATUS_MOVED_PERMANENTLY:   "Moved Permanently",
	STATUS_MOVED_TEMPORARILY:   "Moved Temporarily",
	STATUS_NOT_MODIFIED:        "Not Modified",
	STATUS_BAD_REQUEST:         "Bad Request",
	STATUS_UNAUTHORIZED:        "Unauthorized",
	STATUS_FORBIDDEN:           "Forbidden",
	STATUS_NOT_FOUND:           "Not Found",
	STATUS_INTERNAL_ERROR:      "Internal Error",
	STATUS_NOT_IMPLEMENTED:     "Not Implemented",
	STATUS_BAD_GATEWAY:         "Bad Gateway",
	STATUS_SERVICE_UNAVAILABLE: "Service Unavailable",
}

const KEEP_ALIVE_TIMEOUT = 5

var ErrParsing = errors.New("parsing error")

func isEmpty(element string) bool {
	return element == ""
}

func isURIMatch(requestPath string, pattern string) bool {
	var requestParts = strings.Split(requestPath, "/")
	var patternParts = strings.Split(pattern, "/")

	requestParts = slices.DeleteFunc(requestParts, isEmpty)
	patternParts = slices.DeleteFunc(patternParts, isEmpty)

	if len(requestParts) < len(patternParts) {
		return false
	}

	var j = 0
	for i, part := range patternParts {
		if part == requestParts[j] {
			j += 1
		} else if part == "*" {
			j = len(requestParts) - (len(patternParts) - (i + 1))
		} else if part != requestParts[j] {
			return false
		}
	}

	return j == len(requestParts)
}

func parseBodyWithFullContent(bodyLength int64, bodyReader *textproto.Reader) ([]byte, error) {
	var bodyBuffer []byte = make([]byte, bodyLength)
	readBodyLength, err := io.ReadFull(bodyReader.R, bodyBuffer)
	if err != nil {
		return nil, err
	}

	return bodyBuffer[:readBodyLength], nil
}

func parseServerChunkedBody(bodyReader *textproto.Reader, connection net.Conn, request *ServerHTTPRequest, response *ServerHTTPResponse, onChunk ServerChunkFunction) ([]byte, error) {
	var bodyBytes *bytes.Buffer = new(bytes.Buffer)
	var isFinished = false
	for !isFinished {
		connection.SetReadDeadline(time.Now().Add(KEEP_ALIVE_TIMEOUT * time.Second))
		firstLine, err := bodyReader.ReadLine()
		for err != nil || firstLine == "" {
			firstLine, err = bodyReader.ReadLine()
			if err != nil {
				return nil, err
			}
		}
		firstLine = strings.TrimSpace(firstLine)
		chunkLength, err := strconv.ParseUint(firstLine, 16, 32)
		if err != nil {
			return nil, err
		}
		if chunkLength != 0 {
			var chunkBuffer = make([]byte, chunkLength)
			read, err := io.ReadFull(bodyReader.R, chunkBuffer)
			if err != nil {
				return nil, err
			}
			if onChunk != nil {
				isFinished = !onChunk(chunkBuffer[:read], *request, response)
				bodyBytes.Reset()
			} else {
				bodyBytes.Write(chunkBuffer[:read])
			}
		} else {
			isFinished = true
		}
		bodyReader.ReadLine()
	}
	return bodyBytes.Bytes(), nil
}

func parseClientChunkedBody(bodyReader *textproto.Reader, connection net.Conn, response *ClientHTTPResponse, onChunk ClientChunkFunction) (*bytes.Buffer, error) {
	var bodyBytes *bytes.Buffer = new(bytes.Buffer)
	var isFinished = false
	for !isFinished {
		connection.SetReadDeadline(time.Now().Add(KEEP_ALIVE_TIMEOUT * time.Second))
		firstLine, err := bodyReader.ReadLine()
		for err != nil || firstLine == "" {
			firstLine, err = bodyReader.ReadLine()
			if err != nil {
				return nil, err
			}
		}
		firstLine = strings.TrimSpace(firstLine)
		chunkLength, err := strconv.ParseUint(firstLine, 16, 32)
		if err != nil {
			return nil, err
		}
		if chunkLength != 0 {
			var chunkBuffer = make([]byte, chunkLength)
			read, err := io.ReadFull(bodyReader.R, chunkBuffer)
			if err != nil {
				return nil, err
			}
			if onChunk != nil {
				isFinished = !onChunk(chunkBuffer[:read], response)
				bodyBytes.Reset()
			} else {
				bodyBytes.Write(chunkBuffer[:read])
			}
		} else {
			isFinished = true
		}
		bodyReader.ReadLine()
	}

	return bodyBytes, nil
}

func isClosingRequest(request httpRequest) bool {
	if request.Version() == "1.0" {
		return !request.HasHeaderValue("Connection", "keep-alive")
	} else {
		return request.HasHeaderValue("Connection", "close")
	}
}
