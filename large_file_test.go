package easyhttp

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func TestLargeFiles(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	file, err := os.Open("testdata/lusiadasTest.txt")
	if err != nil {
		fmt.Println(err)
	}

	body, err := io.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}

	request, err := NewRequestWithBody("http://localhost:1234/large", body)
	if err != nil {
		t.Fatal(err.Error())
	}
	request.SetVersion("1.0")
	response, err := client.POST(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
		t.FailNow()
	}

	if !response.HasHeaderValue("Content-Length", "362128") {
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

func TestSmallerContentLength(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	file, err := os.Open("testdata/lusiadasTest.txt")
	if err != nil {
		fmt.Println(err)
	}

	body, err := io.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}

	request, err := NewRequestWithBody("http://localhost:1234/large", body)
	if err != nil {
		t.Fatal(err.Error())
	}
	request.SetVersion("1.0")

	request.SetHeader("Content-Length", "10000")
	response, err := client.POST(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
		t.FailNow()
	}

	if !response.HasHeaderValue("Content-Length", "362128") {
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

func TestBiggerContentLength(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)

	client := NewHTTPClient()
	file, err := os.Open("testdata/lusiadasTest.txt")
	if err != nil {
		fmt.Println(err)
	}

	body, err := io.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}

	request, err := NewRequestWithBody("http://localhost:1234/large", body)
	if err != nil {
		t.Fatal(err.Error())
	}
	request.SetVersion("1.0")
	request.SetHeader("Content-Length", "1000000")
	response, err := client.POST(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
		t.FailNow()
	}

	if !response.HasHeaderValue("Content-Length", "362128") {
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

func TestServerFileUpload(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)
	client := NewHTTPClient()

	request, err := NewRequest("http://localhost:1234/testdata/lusiadasTest.txt")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.SetVersion("1.0")
	response, err := client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.StatusCode != STATUS_OK {
		t.FailNow()
	}

	if !response.HasHeaderValue("Content-Length", "362128") {
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
