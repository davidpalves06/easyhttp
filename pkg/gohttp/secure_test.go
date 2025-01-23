package gohttp

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func setupSecureServer(tb testing.TB) func(tb testing.TB) {
	cert, err := tls.LoadX509KeyPair("testdata/cert.pem", "testdata/key.pem")
	if err != nil {
		os.Exit(1)
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	server, err := NewTLSHTTPServer(":1234", config)
	if err != nil {
		tb.Fatalf("Error creating HTTP Server")
	}

	server.HandleGET("/path", handleRequest)
	server.HandleGET("/", handleRequest)
	server.HandlePOST("/resource", handleRequest)
	server.HandlePOST("/large", handleEcho)
	server.HandleGET("/chunked", handleChunked)
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

func TestHTTPSServer(t *testing.T) {
	tearDown := setupSecureServer(t)
	defer tearDown(t)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", "https://localhost:1234/path", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Connection", "close")

	response, err := client.Do(req)
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
}

func TestHTTPSClient(t *testing.T) {
	client := NewHTTPClient()
	client.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("TestHeader", "Hello")
		w.WriteHeader(STATUS_OK)
		w.Write([]byte("Hello World!\n"))
	}))
	defer server.Close()

	request, err := NewRequest(server.URL + "/path")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.SetVersion("1.0")
	response, err := client.GET(request)
	if err != nil || response == nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_OK {
		t.Fatalf("Wrong Status Code: %d\n", response.StatusCode)
	}

	value := response.GetHeader("TestHeader")
	if value != "Hello" {
		t.Fatalf("Wrong Header : %v\n", value)
	}
	bodyBuffer := make([]byte, 1024)
	bodyLength, _ := response.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Hello World!\n" {
		t.Fatalf("Wrong Body")
	}
}

func TestHTTPSClientServer(t *testing.T) {
	tearDown := setupSecureServer(t)
	defer tearDown(t)
	client := NewHTTPClient()
	client.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	request, err := NewRequest("https://localhost:1234/path")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.version = "1.0"
	response, err := client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.version != "1.0" {
		t.Fatalf("HTTP VERSION IS WRONG")
	}

	headerValue := response.GetHeader("TestHeader")
	if response.StatusCode != STATUS_OK || headerValue != "Hello" {
		t.FailNow()
	}

	bodyBuffer := make([]byte, 1024)
	bodyLength, _ := response.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Hello World!\n" {
		t.FailNow()
	}

	request, err = NewRequest("https://localhost:1234/")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.version = "1.1"
	response, err = client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.version != "1.1" {
		t.Fatalf("HTTP VERSION IS WRONG")
	}

	headerValue = response.GetHeader("TestHeader")
	if response.StatusCode != STATUS_OK || headerValue != "Hello" {
		t.FailNow()
	}

	bodyLength, _ = response.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Hello World!\n" {
		t.FailNow()
	}

	request, err = NewRequest("https://localhost:1234/resource")
	request.SetHeader("Connection", "close")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.version = "2.0"
	response, err = client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.version != "2.0" {
		t.Fatalf("HTTP VERSION IS WRONG")
	}

	if response.StatusCode != STATUS_NOT_IMPLEMENTED {
		t.FailNow()
	}
}
