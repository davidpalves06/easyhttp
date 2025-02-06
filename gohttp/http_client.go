package gohttp

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/url"
	"time"
)

type httpClient struct {
	activeConnections map[string]net.Conn
	TLSConfig         *tls.Config
	MaxRedirects      uint8
	*CookieStorage
}

func NewHTTPClient() httpClient {
	return httpClient{
		activeConnections: make(map[string]net.Conn),
		MaxRedirects:      10,
		CookieStorage:     newCookieStorage(),
	}
}

func (c *httpClient) GET(request ClientHTTPRequest) (*ClientHTTPResponse, error) {
	request.method = MethodGet
	return c.sendRequest(request)
}

func (c *httpClient) HEAD(request ClientHTTPRequest) (*ClientHTTPResponse, error) {
	request.method = MethodHead
	return c.sendRequest(request)
}

func (c *httpClient) POST(request ClientHTTPRequest) (*ClientHTTPResponse, error) {
	request.method = MethodPost
	return c.sendRequest(request)
}

func (c *httpClient) DELETE(request ClientHTTPRequest) (*ClientHTTPResponse, error) {
	request.method = MethodDelete
	return c.sendRequest(request)
}

func (c *httpClient) PUT(request ClientHTTPRequest) (*ClientHTTPResponse, error) {
	request.method = MethodPut
	return c.sendRequest(request)
}

func (c *httpClient) PATCH(request ClientHTTPRequest) (*ClientHTTPResponse, error) {
	request.method = MethodPatch
	return c.sendRequest(request)
}

func (c *httpClient) sendRequest(request ClientHTTPRequest) (*ClientHTTPResponse, error) {
	var response *ClientHTTPResponse
	var redirects uint8 = 0
	var isRedirect = true
	for redirects < c.MaxRedirects && isRedirect {
		if request.uri.Host == "" {
			host, ok := request.headers["host"]
			if !ok {
				return nil, errors.New("no host to send request. Use absolute URI or host header")
			}
			request.uri.Host = host[0]
		}

		request.SetHeader("Host", request.uri.Host)

		var connection net.Conn
		var err error

		connection, exists := c.activeConnections[request.uri.Host]
		if !exists || !checkIfConnectionIsStillOpen(connection) {
			if request.uri.Scheme == "https" {
				connection, err = tls.Dial("tcp", request.uri.Host, c.TLSConfig)
				if err != nil {
					return nil, err
				}
			} else {
				connection, err = net.Dial("tcp", request.uri.Host)
				if err != nil {
					return nil, err
				}
			}
		}

		request.cookies = c.Cookies(request.uri)
		requestBytes, err := request.toBytes()
		if err != nil {
			return nil, err
		}
		_, err = connection.Write(requestBytes)
		if err != nil {
			return nil, err
		}

		if request.chunked {
			request.sendChunks(connection)
		}

		if request.timeout > 0 {
			connection.SetReadDeadline(time.Now().Add(request.timeout))
		} else {
			connection.SetReadDeadline(time.Time{})
		}
		response, err = parseResponse(connection, request)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				connection.Close()
				delete(c.activeConnections, request.uri.Host)
				return nil, ErrClientTimeout
			}
			return nil, err
		}
		c.CookieStorage.SetCookies(request.uri, response.Cookies())

		if isClosingRequest(&request) {
			connection.Close()
			delete(c.activeConnections, request.uri.Host)
		} else {
			c.activeConnections[request.uri.Host] = connection
		}

		if isRedirected(response) {
			var location = response.GetHeader("Location")[0]
			uri, _ := url.ParseRequestURI(location)
			if uri.Host != "" {
				err = request.SetURI(location)
				if err != nil {
					return nil, errors.New("bad redirect location")
				}
			} else {
				request.uri.Path = location
			}
		} else {
			isRedirect = false
		}
		redirects++
	}

	if redirects == c.MaxRedirects {
		return nil, errors.New("too many redirects")
	}

	return response, nil
}

func isRedirected(response *ClientHTTPResponse) bool {
	return response.StatusCode >= 300 && response.StatusCode < 400
}

func checkIfConnectionIsStillOpen(connection net.Conn) bool {
	one := make([]byte, 1)
	connection.SetReadDeadline(time.Now().Add(100 * time.Microsecond))

	if _, err := connection.Read(one); err == io.EOF {
		return false
	} else {
		connection.SetReadDeadline(time.Time{})
		return true
	}
}
