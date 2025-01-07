package main

import (
	"fmt"

	"github.com/davidpalves06/WebSocket/pkg/gohttp"
)

func main() {

	body := "name=FirstName%20LastName&email=bsmth%40example.com"
	request, err := gohttp.NewRequestWithBody("http://localhost:1234/path", []byte(body))
	request.SetHeader("Host", "example.com")
	request.SetHeader("Content-Type", "text/html")

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	response, err := gohttp.POST(request)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	buffer := make([]byte, 1024)
	fmt.Printf("Status Code: %d\n", response.StatusCode)
	read, _ := response.Body.Read(buffer)

	fmt.Println(string(buffer[:read]))
}
