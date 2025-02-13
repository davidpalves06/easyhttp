package easyhttp

import "testing"

func TestRedirects(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)

	client := NewHTTPClient()

	request, err := NewRequest("http://localhost:1234/redirect")
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
}

func TestInfiniteRedirects(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)

	client := NewHTTPClient()

	request, err := NewRequest("http://localhost:1234/infinite/redirect")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.version = "1.0"
	_, err = client.GET(request)
	if err == nil || err.Error() != "too many redirects" {
		t.Fatal("Test should fail because to many redirects")
	}
}
