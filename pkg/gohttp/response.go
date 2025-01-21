package gohttp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/textproto"
	"slices"
	"strconv"
	"strings"
	"time"
)

type HTTPResponse struct {
	headers    Headers
	statusCode int
	version    string
	body       *bytes.Buffer
	conn       net.Conn
	chunked    bool
	method     string
}

func (r *HTTPResponse) Write(p []byte) (n int, err error) {
	return r.body.Write(p)
}

func (r *HTTPResponse) SendChunk() (int, error) {
	if r.method == MethodHead {
		return 0, errors.New("head message cannot be chunked")
	}
	if !r.chunked {
		r.chunked = true
		responseBytes, err := r.toBytes()
		if err != nil {
			return 0, err
		}
		r.conn.Write(responseBytes)
	}

	buffer := new(bytes.Buffer)
	var chunkLength = r.body.Len()
	if chunkLength <= 0 {
		return 0, errors.New("chunk size cannot be 0")
	}
	chunkLengthLine := fmt.Sprintf("%x \r\n", chunkLength)
	buffer.WriteString(chunkLengthLine)
	buffer.Write(r.body.Bytes())

	buffer.WriteString("\r\n")

	r.conn.Write(buffer.Bytes())

	r.body.Reset()
	return chunkLength, nil
}

func (r *HTTPResponse) HasBody() bool {
	return r.body != nil
}

func (r *HTTPResponse) Read(buffer []byte) (int, error) {
	return r.body.Read(buffer)
}

func (r *HTTPResponse) SetHeader(headerName string, headerValue string) {
	r.headers[strings.ToLower(strings.TrimSpace(headerName))] = strings.TrimSpace(headerValue)
}

func (r *HTTPResponse) SetStatus(status int) {
	r.statusCode = status
}

func (r *HTTPResponse) GetHeader(key string) string {
	value, found := r.headers[strings.ToLower(key)]
	if found {
		return value
	} else {
		return ""
	}
}

func (r *HTTPResponse) ExistsHeader(key string) bool {
	_, found := r.headers[strings.ToLower(key)]
	return found
}

func (r *HTTPResponse) Headers() Headers {
	return r.headers
}

func (r *HTTPResponse) toBytes() ([]byte, error) {
	buffer := new(bytes.Buffer)
	var reasonPhrase = reasons[r.statusCode]
	var statusLine = fmt.Sprintf("HTTP/%s %d %s\r\n", r.version, r.statusCode, reasonPhrase)
	buffer.WriteString(statusLine)

	addEssentialHTTPHeaders(r)

	if r.chunked {
		r.SetHeader("Transfer-Encoding", "chunked")
	} else if r.body != nil && r.body.Len() > 0 {
		r.SetHeader("Content-Length", strconv.Itoa(r.body.Len()))
	}

	for headerName, headerValue := range r.headers {
		var headerLine = fmt.Sprintf("%s: %s\r\n", headerName, headerValue)
		buffer.WriteString(headerLine)
	}

	buffer.WriteString("\r\n")

	contentLengthValue := r.GetHeader("Content-Length")
	if contentLengthValue != "" && !r.chunked {
		bodyLength, err := strconv.ParseInt(contentLengthValue, 10, 32)
		if err != nil || bodyLength == 0 {
			return nil, errors.New("content length not valid")
		}

		bodyBuffer := make([]byte, 2048)
		var readSize int64

		for readSize < bodyLength {
			read, err := r.body.Read(bodyBuffer)
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

func newBadRequestResponse() HTTPResponse {
	badRequestResponse := HTTPResponse{
		version:    "1.0",
		statusCode: 400,
	}
	return badRequestResponse
}

func addEssentialHTTPHeaders(response *HTTPResponse) {

	response.SetHeader("Date", time.Now().UTC().Format(time.RFC1123))
	response.SetHeader("Server", softwareName)

	if response.statusCode == STATUS_UNAUTHORIZED {
		if _, exists := response.headers["WWW-Authenticate"]; !exists {
			log.Printf("Warning : Status 401 has no WWW-Authenticate header\n")
		}
	}
}

func parseResponseStatusLine(statusLine string, response *HTTPResponse) error {
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
	response.statusCode = int(parsedStatus)

	return nil
}

func parseResponseHeaders(responseReader *textproto.Reader, response *HTTPResponse) {
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

func parseResponsefromConnection(connection net.Conn) (*HTTPResponse, error) {
	var responseReader = textproto.NewReader(bufio.NewReader(connection))
	var response = &HTTPResponse{
		headers: make(Headers),
		conn:    connection,
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

	contentLengthValue := response.GetHeader("Content-Length")

	if contentLengthValue != "" {
		var bodyLength, err = strconv.ParseInt(contentLengthValue, 10, 32)
		if err != nil {
			return nil, ErrParsing
		}
		if bodyLength != 0 {
			responseBytes, err := parseBodyWithFullContent(bodyLength, responseReader)
			if err != nil {
				return nil, err
			}
			response.body = bytes.NewBuffer(responseBytes)
		}
	} else {
		response.body = nil
	}

	return response, nil
}
