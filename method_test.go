package easyhttp

import (
	"testing"
)

func TestMethods(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequest("http://localhost:1234/path")
	if err != nil {
		t.Fatal(err.Error())
	}
	response, err := client.DELETE(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
		t.FailNow()
	}

	bodyBuffer := make([]byte, 1024)
	bodyLength, _ := response.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Hello World!\n" {
		t.FailNow()
	}

	request, err = NewRequest("http://localhost:1234/path")
	if err != nil {
		t.Fatal(err.Error())
	}
	response, err = client.PUT(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
		t.FailNow()
	}

	bodyBuffer = make([]byte, 1024)
	bodyLength, _ = response.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Hello World!\n" {
		t.FailNow()
	}

	request, err = NewRequest("http://localhost:1234/path")
	request.CloseConnection()
	if err != nil {
		t.Fatal(err.Error())
	}
	response, err = client.PATCH(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
		t.FailNow()
	}

	bodyBuffer = make([]byte, 1024)
	bodyLength, _ = response.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Hello World!\n" {
		t.FailNow()
	}
}
