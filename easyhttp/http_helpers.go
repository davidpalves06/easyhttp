//Package easyhttp implements the HTTP Protocol to create a server to receive requests or client to send requests.
//
// The easyhttp package should be used only on HTTP 1.1 for now. It makes HTTP easy by providing simple methods
// so even if the user does not know the full HTTP protocol, he can still implement HTTP communication easily.

package easyhttp

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

// Interface that HTTP Requests should follow
type httpRequest interface {
	SetHeader(key string, value string)
	GetHeader(key string) []string
	Version() string
	SetVersion(version string) error
	HasHeaderValue(key string, value string) bool
}

var mime_types = map[string]string{
	"txt":  "text/plain",
	"html": "text/html",
	"css":  "text/css",
	"js":   "application/javascript",
	"json": "application/json",
	"pdf":  "application/pdf",
	"jpg":  "image/jpeg",
	"png":  "image/png",
	"gif":  "image/gif",
	"mp4":  "video/mp4",
	"zip":  "application/zip",
	"svg":  "image/svg+xml",
}

// HTTP Headers
type Headers map[string][]string

// Software Agent Name
const softwareName = "Easyhttp 1.0"

// HTTP Methods
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

var validMethods = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"}
var validVersions = []string{"1.0", "1.1"}

var ErrInvalidLength = errors.New("invalid content length")
var ErrInvalidMethod = errors.New("invalid method")
var ErrMethodNotAllowed = errors.New("method not allowed")
var ErrNotFound = errors.New("not found")
var ErrVersionNotSupported = errors.New("version not supported")
var ErrBadRequest = errors.New("bad request")
var ErrRequestTimeout = errors.New("request timeout")
var ErrClientTimeout = errors.New("client timeout")
var ErrInternalError = errors.New("internal error")

// HTTP Status
const (
	STATUS_CONTINUE                      = 100
	STATUS_SWITCHING_PROTOCOL            = 101
	STATUS_OK                            = 200
	STATUS_CREATED                       = 201
	STATUS_ACCEPTED                      = 202
	STATUS_NON_AUTHORATIVE_INFORMATION   = 203
	STATUS_NO_CONTENT                    = 204
	STATUS_RESET_CONTENT                 = 205
	STATUS_PARTIAL_CONTENT               = 206
	STATUS_MULTIPLE_CHOICES              = 300
	STATUS_MOVED_PERMANENTLY             = 301
	STATUS_FOUND                         = 302
	STATUS_SEE_OTHER                     = 303
	STATUS_NOT_MODIFIED                  = 304
	STATUS_USE_PROXY                     = 305
	STATUS_UNUSED                        = 306
	STATUS_TEMPORARY_REDIRECT            = 307
	STATUS_PERMANENT_REDIRECT            = 308
	STATUS_BAD_REQUEST                   = 400
	STATUS_UNAUTHORIZED                  = 401
	STATUS_PAYMENT_REQUIRED              = 402
	STATUS_FORBIDDEN                     = 403
	STATUS_NOT_FOUND                     = 404
	STATUS_METHOD_NOT_ALLOWED            = 405
	STATUS_NOT_ACCEPTABLE                = 406
	STATUS_PROXY_AUTHENTICATION_REQUIRED = 407
	STATUS_REQUEST_TIMEOUT               = 408
	STATUS_CONFLICT                      = 409
	STATUS_GONE                          = 410
	STATUS_LENGTH_REQUIRED               = 411
	STATUS_PRECONDITION_FAILED           = 412
	STATUS_CONTENT_TOO_LARGE             = 413
	STATUS_URI_TOO_LONG                  = 414
	STATUS_UNSUPPORTED_MEDIA_TYPE        = 415
	STATUS_RANGE_NOT_SATISFIABLE         = 416
	STATUS_MISDIRECTED_REQUEST           = 421
	STATUS_UNPROCESSABLE_CONTENT         = 422
	STATUS_UPGRADE_REQUIRED              = 426
	STATUS_INTERNAL_ERROR                = 500
	STATUS_NOT_IMPLEMENTED               = 501
	STATUS_BAD_GATEWAY                   = 502
	STATUS_SERVICE_UNAVAILABLE           = 503
	STATUS_GATEWAY_TIMEOUT               = 504
	STATUS_HTTP_VERSION_NOT_SUPPORTED    = 505
)

var reasons = map[int]string{
	STATUS_CONTINUE:                      "Continue",
	STATUS_SWITCHING_PROTOCOL:            "Switching Protocol",
	STATUS_OK:                            "OK",
	STATUS_CREATED:                       "Created",
	STATUS_ACCEPTED:                      "Accepted",
	STATUS_NON_AUTHORATIVE_INFORMATION:   "Non Authorative Information",
	STATUS_NO_CONTENT:                    "No Content",
	STATUS_RESET_CONTENT:                 "Reset Content",
	STATUS_PARTIAL_CONTENT:               "Partial Content",
	STATUS_MULTIPLE_CHOICES:              "Multiple Choices",
	STATUS_MOVED_PERMANENTLY:             "Moved Permanently",
	STATUS_FOUND:                         "Found",
	STATUS_SEE_OTHER:                     "See Other",
	STATUS_NOT_MODIFIED:                  "Not Modified",
	STATUS_USE_PROXY:                     "Use Proxy",
	STATUS_TEMPORARY_REDIRECT:            "Temporary Redirect",
	STATUS_PERMANENT_REDIRECT:            "Permanent Redirect",
	STATUS_BAD_REQUEST:                   "Bad Request",
	STATUS_UNAUTHORIZED:                  "Unauthorized",
	STATUS_PAYMENT_REQUIRED:              "Payment Required",
	STATUS_FORBIDDEN:                     "Forbidden",
	STATUS_NOT_FOUND:                     "Not Found",
	STATUS_METHOD_NOT_ALLOWED:            "Method Not Allowed",
	STATUS_NOT_ACCEPTABLE:                "Not Acceptable",
	STATUS_PROXY_AUTHENTICATION_REQUIRED: "Proxy Authentication Required",
	STATUS_REQUEST_TIMEOUT:               "Request Timeout",
	STATUS_CONFLICT:                      "Conflict",
	STATUS_GONE:                          "Gone",
	STATUS_LENGTH_REQUIRED:               "Length Required",
	STATUS_PRECONDITION_FAILED:           "Precondition Failed",
	STATUS_CONTENT_TOO_LARGE:             "Content Too Large",
	STATUS_URI_TOO_LONG:                  "URI Too Long",
	STATUS_UNSUPPORTED_MEDIA_TYPE:        "Unsupported Media Type",
	STATUS_RANGE_NOT_SATISFIABLE:         "Range Not Satisfiable",
	STATUS_MISDIRECTED_REQUEST:           "Misdirected Request",
	STATUS_UNPROCESSABLE_CONTENT:         "Unprocessable Content",
	STATUS_UPGRADE_REQUIRED:              "Upgrade Required",
	STATUS_INTERNAL_ERROR:                "Internal Error",
	STATUS_NOT_IMPLEMENTED:               "Not Implemented",
	STATUS_BAD_GATEWAY:                   "Bad Gateway",
	STATUS_SERVICE_UNAVAILABLE:           "Service Unavailable",
	STATUS_GATEWAY_TIMEOUT:               "Gateway Timeout",
	STATUS_HTTP_VERSION_NOT_SUPPORTED:    "HTTP Version Not Supported",
}

const KEEP_ALIVE_TIMEOUT = 5

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
