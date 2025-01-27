package gohttp

import (
	"testing"
	"time"
)

func TestVersion(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequest("http://localhost:1234/path")
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

	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
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
	request.CloseConnection()
	response, err = client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.version != "1.1" {
		t.Fatalf("HTTP VERSION IS WRONG")
	}

	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
		t.FailNow()
	}

	bodyLength, _ = response.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Hello World!\n" {
		t.FailNow()
	}

}

func TestHeadRequests(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequest("http://localhost:1234/path")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.AddHeader("Connection", "close")
	response, err := client.HEAD(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
		t.FailNow()
	}
	if response.HasBody() {
		t.Fatalf("Body is not empty\n")
	}

}

func TestMultipleHeaderRequests(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequest("http://localhost:1234/path")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.AddHeader("Connection", "close")
	request.AddHeader("RequestHeader", "Test")
	request.AddHeader("RequestHeader", "Passed")

	response, err := client.HEAD(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("ResponseHeader", "Test") || !response.HasHeaderValue("ResponseHeader", "Passed") || len(response.GetHeader("ResponseHeader")) < 2 {
		t.Fatalf("Status code or header are wrong")
	}
	if response.HasBody() {
		t.Fatalf("Body is not empty\n")
	}

}

func TestServerClosedPermanentConnection(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequest("http://localhost:1234/")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.version = "1.1"
	response, err := client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.version != "1.1" {
		t.Fatal("HTTP VERSION IS WRONG")
	}

	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
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
	if err != nil {
		t.Fatal(err.Error())
	}
	request.version = "1.1"
	request.CloseConnection()
	response, err = client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.version != "1.1" {
		t.Fatalf("HTTP VERSION IS WRONG")
	}

	if response.StatusCode != STATUS_NOT_IMPLEMENTED {
		t.FailNow()
	}

}
