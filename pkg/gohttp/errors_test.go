package gohttp

import (
	"testing"
)

func TestInvalidLength(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequest("http://localhost:1234/path")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.version = "1.0"
	request.SetHeader("Content-Length", "avc")
	response, err := client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.version != "1.0" {
		t.Fatalf("HTTP VERSION IS WRONG")
	}

	if response.StatusCode != STATUS_LENGTH_REQUIRED {
		t.FailNow()
	}

}

func TestInvalidMethod(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequest("http://localhost:1234/path")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.version = "1.0"
	request.method = "AVC"
	response, err := client.sendRequest(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.version != "1.0" {
		t.Fatalf("HTTP VERSION IS WRONG")
	}

	if response.StatusCode != STATUS_METHOD_NOT_ALLOWED {
		t.FailNow()
	}

}

func TestUnsupportedVersion(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequest("http://localhost:1234/path")
	if err != nil {
		t.Fatal(err.Error())
	}

	request.version = "2.0"
	response, err := client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_HTTP_VERSION_NOT_SUPPORTED {
		t.FailNow()
	}

}

func TestPanicOnHandler(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequest("http://localhost:1234/panic")
	if err != nil {
		t.Fatal(err.Error())
	}

	response, err := client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_INTERNAL_ERROR {
		t.FailNow()
	}

}
