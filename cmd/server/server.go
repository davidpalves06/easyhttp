package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

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
	server, err := gohttp.NewHTTPServer(":1234")
	if err != nil {
		fmt.Println("Error creating HTTP Server")
		return
	}

	server.HandleGET("/path", handleRequest)
	server.HandlePOST("/path", handleRequest)
	server.HandleGET("/", handleRequestTwo)

	go func() {
		log.Println("Starting server")
		server.Run()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	<-sigChan
	server.Close()

}
