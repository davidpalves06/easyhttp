package easyhttp

import (
	"testing"
)

func TestCookies(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)

	client := NewHTTPClient()
	request, err := NewRequest("http://localhost:1234/path")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.AddHeader("Connection", "close")
	var cookie *Cookie = &Cookie{
		Name:     "TestID",
		Value:    "12345",
		Path:     "/",
		Domain:   "localhost",
		MaxAge:   3600,
		Secure:   false,
		HTTPOnly: true,
		SameSite: SAME_SITE_LAX,
	}

	client.SetCookies(request.uri, []*Cookie{cookie})
	response, err := client.HEAD(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.StatusCode != STATUS_OK {
		t.Fatalf("Status code wrong")
	}

	if !response.HasHeaderValue("CookieTest", "Pass") {
		t.Fatalf("Wrong header")
	}

	if response.HasBody() {
		t.Fatalf("Body is not empty\n")
	}
}

func TestCookieServerResponse(t *testing.T) {
	tearDown := setupServer(t)
	defer tearDown(t)

	client := NewHTTPClient()
	request, err := NewRequest("http://localhost:1234/cookie")
	if err != nil {
		t.Fatal(err.Error())
	}

	response, err := client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.StatusCode != STATUS_OK {
		t.Fatalf("Status code wrong")
	}

	if len(response.Cookies()) != 1 {
		t.Fatalf("Wrong cookies")
	}

	bodyBuffer := make([]byte, 1024)
	bodyLength, _ := response.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Sent cookie!\n" {
		t.FailNow()
	}

	request, err = NewRequest("http://localhost:1234/cookie")
	if err != nil {
		t.Fatal(err.Error())
	}
	request.AddHeader("Connection", "close")
	response, err = client.GET(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	if response.StatusCode != STATUS_OK {
		t.Fatalf("Status code wrong")
	}

	if len(response.Cookies()) != 0 {
		t.Fatalf("Wrong cookies")
	}

	bodyBuffer = make([]byte, 1024)
	bodyLength, _ = response.Read(bodyBuffer)
	if string(bodyBuffer[:bodyLength]) != "Cookie Received!\n" {
		t.FailNow()
	}
}
