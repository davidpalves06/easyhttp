package gohttp

type Headers map[string]string

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
