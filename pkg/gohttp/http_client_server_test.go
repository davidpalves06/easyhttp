package gohttp

import (
	"testing"
	"time"
)

func TestVersion(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)

	request, err := NewRequest("http://localhost:1234/path")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.version = "1.0"
	response, err := GET(request)
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

	request, err = NewRequest("http://localhost:1234/")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.version = "1.1"
	response, err = GET(request)
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

	request, err = NewRequest("http://localhost:1234/resource")
	request.SetHeader("Connection", "close")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.version = "2.0"
	response, err = GET(request)
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

func TestHeadRequests(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)

	request, err := NewRequest("http://localhost:1234/path")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.headers["Connection"] = "close"
	response, err := HEAD(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	headerValue := response.GetHeader("TestHeader")
	if response.StatusCode != STATUS_OK || headerValue != "Hello" {
		t.FailNow()
	}
	if response.HasBody() {
		t.Fatalf("Body is not empty\n")
	}

}

func TestServerClosedPermanentConnection(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)

	request, err := NewRequest("http://localhost:1234/")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.version = "1.1"
	response, err := GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.version != "1.1" {
		t.Fatal("HTTP VERSION IS WRONG")
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
	//w8 for connection to be closed by server
	time.Sleep(6 * time.Second)

	request, err = NewRequest("http://localhost:1234/resource")
	request.SetHeader("Connection", "close")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.version = "2.0"
	response, err = GET(request)
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
