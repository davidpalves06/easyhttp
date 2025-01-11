package main

import (
	"fmt"
	"time"

	"github.com/davidpalves06/WebSocket/pkg/gohttp"
)

func main() {

	// body := "name=FirstName%20LastName&email=bsmth%40example.com"
	request, err := gohttp.NewRequest("http://localhost:1234/path")
	request.SetHeader("Host", "example.com")
	request.SetHeader("Content-Type", "text/html")

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for {

		response, err := gohttp.POST(request)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		buffer := make([]byte, 1024)
		fmt.Printf("Status Code: %d\n", response.StatusCode)
		if response.Body != nil {
			read, err := response.Body.Read(buffer)
			if err != nil {
				fmt.Printf("%s\n", err.Error())
			} else {
				fmt.Println(string(buffer[:read]))
			}
		}
		time.Sleep(5 * time.Second)
	}
}
