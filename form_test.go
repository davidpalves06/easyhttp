package easyhttp

import "testing"

func TestForm(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)

	client := NewHTTPClient()
	var body string = "test=test&next=before"
	request, err := NewRequestWithBody("http://localhost:1234/form", []byte(body))
	if err != nil {
		t.Fatal(err.Error())
	}
	request.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	request.version = "1.1"
	response, err := client.POST(request)
	if err != nil {
		t.Fatal(err.Error())
	}

	if response.version != "1.1" {
		t.Fatal("HTTP VERSION IS WRONG")
	}

	if response.StatusCode != STATUS_OK || !response.HasHeaderValue("TestHeader", "Hello") {
		t.FailNow()
	}
}
