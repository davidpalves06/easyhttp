package easyhttp

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func TestChunkedTransfer(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequestWithBody("http://localhost:1234/large", []byte("This should be ignored"))
	if err != nil {
		t.Fatal(err.Error())
	}

	request.CloseConnection()
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

	response, err := client.POST(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
		t.Fatalf("Status Code or Header Wrong")
	}
	headerLengthHeader := response.GetHeader("Content-Length")
	if headerLengthHeader[len(headerLengthHeader)-1] != "362128" {
		t.Fatalf("Body length is incorrect")
	}

	bodyBuffer := make([]byte, 1024)
	var totalRead int
	for {
		read, err := response.Read(bodyBuffer)
		if err != nil {
			break
		}
		totalRead += read
	}
	if totalRead != 362128 {
		t.Fatalf("Bad body")
	}
}

func TestChunkedResponse(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequest("http://localhost:1234/chunked")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.CloseConnection()
	response, err := client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
		t.Fatalf("Wrong status or header")
	}

	bodyBuffer := make([]byte, 1024)
	var totalRead int
	for {
		read, err := response.Read(bodyBuffer)
		if err != nil {
			break
		}
		totalRead += read
	}
	if totalRead != 362128 {
		t.Fatalf("Bad body")
	}
}

func TestChunkedServerHandlingWithResponseAfterChunks(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequestWithBody("http://localhost:1234/runafter", []byte("This should be ignored"))
	if err != nil {
		t.Fatal(err.Error())
	}

	request.CloseConnection()
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

	response, err := client.POST(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
		t.Fatalf("Wrong status or header")
	}

	bodyBuffer := make([]byte, 1024)
	bodyLength, _ := response.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Hello World!\n" {
		t.FailNow()
	}
}

func TestChunkedServerHandlingWithoutResponseAfterChunks(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequestWithBody("http://localhost:1234/notrun", []byte("This should be ignored"))
	if err != nil {
		t.Fatal(err.Error())
	}

	request.CloseConnection()
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

	response, err := client.POST(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.StatusCode != STATUS_NO_CONTENT || response.GetHeader("TestHeader") != nil {
		t.Fatalf("Wrong status or header")
	}

	if response.HasBody() {
		t.Fatalf("Body should not be present")
	}
}

var total = 0

func handleResponseChunk(chunk []byte, response *ClientHTTPResponse) bool {
	total += len(chunk)
	return true
}

func TestChunkedResponseWithHandlingOnEachChunk(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	total = 0
	request, err := NewRequest("http://localhost:1234/chunked")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.CloseConnection()
	request.onResponseChunk = handleResponseChunk
	response, err := client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
		t.Fatalf("Wrong status or header")
	}

	if response.HasBody() {
		t.Fatalf("Body should not be present")
	}

	if total != 362128 {
		t.Fatalf("Chunk calculation is wrong")
	}
}
