package gohttp

import (
	"bytes"
	"log"
	"net/http"
	"testing"
)

func handleRequest(request HTTPRequest, response *HTTPResponseWriter) {
	response.SetStatus(STATUS_OK)
	response.SetHeader("TestHeader", "Hello")
	response.Write([]byte("Hello World!\n"))
}

func setupServer(tb testing.TB) func(tb testing.TB) {
	server, err := NewHTTPServer(":1234")
	if err != nil {
		tb.Fatalf("Error creating HTTP Server")
	}

	server.HandleGET("/path", handleRequest)
	server.HandleGET("/", handleRequest)
	server.HandlePOST("/resource", handleRequest)
	go func() {
		log.Println("Starting")
		server.Run()
	}()

	// Return a function to teardown the test
	return func(tb testing.TB) {
		log.Println("Closing")
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

	response, err = http.Get("http://localhost:1234/resource")
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
	response, err := http.Post("http://localhost:1234/resource", "plaintext", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Error(err.Error())
	}
	if response.StatusCode != STATUS_OK || response.Header.Get("TestHeader") != "Hello" {
		log.Println("1")
		t.FailNow()
	}
	bodyLength, _ := response.Body.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Hello World!\n" {
		log.Println("1")
		t.FailNow()
	}
}
