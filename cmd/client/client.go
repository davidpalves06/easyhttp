package main

import (
	"fmt"

	"github.com/davidpalves06/WebSocket/pkg/gohttp"
)

func main() {

	request, err := gohttp.NewRequest("http://localhost:1234/path")
	request.SetHeader("Host", "example.com")
	request.SetHeader("Content-Type", "text/html")

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	response, err := gohttp.GET(request)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	buffer := make([]byte, 1024)
	fmt.Printf("Status Code: %d\n", response.StatusCode)
	var totalRead int
	for response.Body != nil {
		read, err := response.Body.Read(buffer)
		if err != nil {
			break
		}
		totalRead += read
	}
	fmt.Println(totalRead)
}
