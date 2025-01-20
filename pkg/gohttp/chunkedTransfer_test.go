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

	request, err := NewRequestWithBody("http://localhost:1234/large", []byte("This should be ignored"))
	if err != nil {
		t.Fatal(err.Error())
	}
	request.headers["Connection"] = "close"
	request.Chunked()

	go func() {
		file, err := os.Open("testdata/lusiadasTest.txt")
		if err != nil {
			fmt.Println(err)
		}

		var chunkBuffer = make([]byte, 4096)
		for {
			read, err := io.ReadFull(file, chunkBuffer)
			if err == io.EOF {
				break
			}
			request.SendChunk(chunkBuffer[:read])
		}
		request.Done()

	}()

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
