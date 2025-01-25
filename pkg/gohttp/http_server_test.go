package gohttp

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
)

func handleRequest(request ServerHTTPRequest, response *ServerHTTPResponse) {
	response.SetStatus(STATUS_OK)
	response.SetHeader("TestHeader", "Hello")
	response.SetHeader("ResponseHeader", "Test")
	response.AddHeader("ResponseHeader", "Passed")
	response.Write([]byte("Hello World!\n"))
}

func handleEcho(request ServerHTTPRequest, response *ServerHTTPResponse) {
	bodyBuffer := make([]byte, 1024)
	buffer := new(bytes.Buffer)
	bodyReader := bytes.NewReader(request.Body)
	var totalRead int
	for {
		read, err := bodyReader.Read(bodyBuffer)
		if err != nil {
			break
		}
		buffer.Write(bodyBuffer)
		totalRead += read
	}
	response.Write(buffer.Bytes()[:totalRead])
	response.SetStatus(STATUS_OK)
	response.SetHeader("TestHeader", "Hello")
}

func handleChunked(request ServerHTTPRequest, response *ServerHTTPResponse) {

	file, err := os.Open("testdata/lusiadasTest.txt")
	if err != nil {
		fmt.Println(err)
	}
	response.SetStatus(STATUS_OK)
	response.SetHeader("TestHeader", "Hello")

	var chunkBuffer = make([]byte, 4096)
	for {
		read, err := io.ReadFull(file, chunkBuffer)
		if err == io.EOF {
			break
		}
		response.Write(chunkBuffer[:read])
		response.SendChunk()
	}
}

func handleChunk(chunk []byte, request ServerHTTPRequest, response *ServerHTTPResponse) bool {
	response.SetStatus(204)
	response.SetHeader("CHUNK", "YES")

	return true
}

func handleInfiniteRedirect(request ServerHTTPRequest, response *ServerHTTPResponse) {
	response.SetStatus(STATUS_MOVED_PERMANENTLY)
	response.SetHeader("Location", "http://localhost:1234/infinite/redirect")
}

func setupServer(tb testing.TB) func(tb testing.TB) {
	server, err := NewHTTPServer(":1234")
	if err != nil {
		tb.Fatalf("Error creating HTTP Server")
	}

	server.HandleGET("/path", handleRequest)
	server.HandleGET("/", handleRequest)
	server.HandlePOST("/resource", handleRequest)
	server.HandlePOST("/large", handleEcho)
	server.HandleGET("/chunked", handleChunked)
	server.HandleGET("/redirect", PermaRedirect("http://localhost:1234/path"))
	server.HandleGET("/infinite/redirect", handleInfiniteRedirect)
	server.HandleGET("/testdata/lusiadasTest.txt", FileServer("testdata"))
	server.HandlePOSTWithOptions("/runafter", handleRequest, HandlerOptions{onChunk: handleChunk, runAfterChunks: true})
	server.HandlePOSTWithOptions("/notrun", handleRequest, HandlerOptions{onChunk: handleChunk, runAfterChunks: false})
	go func() {
		server.Run()
	}()

	return func(tb testing.TB) {
		server.Close()
	}
}
func TestServerGet(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)

	response, err := http.Get("http://localhost:1234/path")
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.StatusCode != STATUS_OK || response.Header.Get("TestHeader") != "Hello" {
		t.FailNow()
	}

	bodyBuffer := make([]byte, 1024)
	bodyLength, _ := response.Body.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Hello World!\n" {
		t.FailNow()
	}

	response, err = http.Get("http://localhost:1234/")
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.StatusCode != STATUS_OK || response.Header.Get("TestHeader") != "Hello" {
		t.FailNow()
	}
	bodyLength, _ = response.Body.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Hello World!\n" {
		t.FailNow()
	}

	request, err := http.NewRequest(MethodGet, "http://localhost:1234/resource", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	request.Header.Add("Connection", "close")

	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.StatusCode != STATUS_NOT_IMPLEMENTED {
		t.FailNow()
	}

}

func TestServerPost(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)

	bodyBuffer := make([]byte, 1024)

	body := "name=FirstName%20LastName&email=bsmth%40example.com"
	request, err := http.NewRequest(MethodPost, "http://localhost:1234/resource", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err.Error())
	}
	request.Header.Add("Connection", "close")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Error(err.Error())
	}
	if response.StatusCode != STATUS_OK || response.Header.Get("TestHeader") != "Hello" {
		t.FailNow()
	}
	bodyLength, _ := response.Body.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Hello World!\n" {
		t.FailNow()
	}
}
