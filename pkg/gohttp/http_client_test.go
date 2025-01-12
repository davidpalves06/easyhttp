package gohttp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("TestHeader", "Hello")
		w.WriteHeader(STATUS_OK)
		w.Write([]byte("Hello World!\n"))
	}))
	defer server.Close()

	request, err := NewRequest(server.URL + "/path")
	if err != nil {
		t.Fatal(err.Error())
	}

	response, err := GET(request)
	if err != nil || response == nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_OK {
		t.Fatalf("Wrong Status Code: %d\n", response.StatusCode)
	}

	value, exists := response.GetHeader("TestHeader")
	if !exists || value != "Hello" {
		t.Fatalf("Wrong Header : %v\n", exists)
	}
	bodyBuffer := make([]byte, 1024)
	bodyLength, _ := response.Body.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Hello World!\n" {
		t.Fatalf("Wrong Body")
	}
}

func TestClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		body, _ := io.ReadAll(r.Body)
		w.Header().Set("TestHeader", "Hello")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	request, err := NewRequestWithBody(server.URL+"/resource", []byte("Echo me this"))
	if err != nil {
		t.Fatal(err.Error())
	}
	response, err := POST(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_OK {
		t.Fatalf("Wrong Status code")
	}
	value, exists := response.GetHeader("TestHeader")
	if !exists || value != "Hello" {
		t.Fatalf("Wrong Header : %v\n", exists)
	}
	bodyBuffer := make([]byte, 1024)
	bodyLength, _ := response.Body.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Echo me this" {
		t.Fatalf("Wrong Body")
	}
}
