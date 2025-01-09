package main

import (
	"fmt"

	"github.com/davidpalves06/WebSocket/pkg/gohttp"
)

func handleRequest(request gohttp.HTTPRequest, response *gohttp.HTTPResponseWriter) {
	bodyBuffer := make([]byte, 1024)
	fmt.Println("Dealing with request")
	if request.Body != nil {
		read, _ := request.Body.Read(bodyBuffer)
		response.Write(bodyBuffer[:read])
	}
	response.SetStatus(gohttp.STATUS_ACCEPTED)
	response.SetHeader("TestHeader", "Hello")
}

func main() {
	fmt.Println("Starting TCP SOCKET")
	socket, err := gohttp.CreateHTTPServer(":1234")
	if err != nil {
		fmt.Println("Error creating socket")
	}

	for {
		err := socket.HandleRequest(handleRequest)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("TCP connection accepted")
	}

}
