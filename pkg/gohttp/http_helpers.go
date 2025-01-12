package gohttp

import (
	"slices"
	"strings"
)

type Headers map[string]string

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
