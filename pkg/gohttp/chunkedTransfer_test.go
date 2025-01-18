package gohttp

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func TestChunkedTransfer(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)

	file, err := os.Open("testdata/lusiadasTest.txt")
	if err != nil {
		fmt.Println(err)
	}

	body, err := io.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}

	request, err := NewRequestWithBody("http://localhost:1234/large", body)
	if err != nil {
		t.Fatal(err.Error())
	}
	// request.headers["Transfer-Encoding"] = "chunked"
	request.headers["Connection"] = "close"
	response, err := POST(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	headerValue := response.GetHeader("TestHeader")
	if response.StatusCode != STATUS_OK || headerValue != "Hello" {
		t.FailNow()
	}
	headerLength := response.GetHeader("Content-Length")
	if headerLength != "362128" {
		t.Fatalf("Body length is incorrect")
	}

	bodyBuffer := make([]byte, 1024)
	var totalRead int
	for {
		read, err := response.Body.Read(bodyBuffer)
		if err != nil {
			break
		}
		totalRead += read
	}
	if totalRead != 362128 {
		t.Fatalf("Bad body")
	}
}
