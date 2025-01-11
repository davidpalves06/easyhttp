package main

import (
	"fmt"

	"github.com/davidpalves06/WebSocket/pkg/gohttp"
)

func handleRequest(request gohttp.HTTPRequest, response *gohttp.HTTPResponseWriter) {
	fmt.Println("Dealing with request")
	response.Write([]byte("Hello World!\n"))
	response.SetStatus(gohttp.STATUS_ACCEPTED)
	response.SetHeader("TestHeader", "Hello")
}

func handleRequestTwo(request gohttp.HTTPRequest, response *gohttp.HTTPResponseWriter) {
	fmt.Println("Dealing with request")
	response.Write([]byte("Hello From Another Path!\n"))
	response.SetStatus(gohttp.STATUS_OK)
}

func main() {
	fmt.Println("Starting TCP SOCKET")
	server, err := gohttp.CreateHTTPServer(":1234")
	if err != nil {
		fmt.Println("Error creating socket")
	}

	server.HandleGET("/path", handleRequest)
	server.HandleGET("/", handleRequestTwo)
	for {

		err := server.HandleRequest()
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("TCP connection accepted")
	}

}
