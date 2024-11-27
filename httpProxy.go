package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/fatih/color"
)

type InterceptConfig struct {
	ConfigName     string          `json:"configName"`
	InterceptLinks []InterceptLink `json:"interceptLinks"`
}

type InterceptLink struct {
	Url       string `json:"url"`
	Intercept bool   `json:"intercept"`
}

var interceptConfig InterceptConfig = InterceptConfig{
	ConfigName:     "default",
	InterceptLinks: []InterceptLink{},
}

func initConfig() {
	file, err := os.Open("interceptLinks.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()
	jsonBytes, _ := io.ReadAll(file)
	json.Unmarshal(jsonBytes, &interceptConfig)
}

func printRequest(data []byte, id uint64) {
	// Print the request data
	fmt.Printf("\nRequest %d data:\n", id)
	color.Cyan(string(data))
	fmt.Println()
}

func handleHttpConn(conn net.Conn, targetAddr net.Addr) {
	defer conn.Close()
	//Create mock http response:
	response := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/html\r\n" +
		"Content-Length: 11\r\n" +
		"\r\n" +
		"Hello World" +
		"\r\n"

	// Increment the counter for each http request
	counter++

	ready := make(chan struct{})
	var requestBuffer bytes.Buffer
	go func() {
		_, err := io.Copy(&requestBuffer, conn)
		if err != nil {
			log.Printf("Failed to read request data: %v", err)
			requestBuffer.Reset()
		} else {
			// Print the request data
			printRequest(requestBuffer.Bytes(), counter)

		}
		ready <- struct{}{}
	}()

	_, err2 := io.Copy(conn, strings.NewReader(response))

	if err2 != nil {
		log.Printf("Failed to write response data: %v", err2)
	}

	<-ready

}
