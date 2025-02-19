package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

var counter uint64 = 0

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

type HttpPrintingProxy struct {
	HttpProxy
}

func (h *HttpPrintingProxy) HandleHttp(conn net.Conn, targetAddr net.Addr) {
	handleHttpConn(conn, targetAddr)
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
	color.Yellow("Intercept config loaded: %v\n", interceptConfig)
}

/*
Debug function to print the request data to the console
*/
func printRequest(r io.Reader, id uint64, ready chan struct{}) {
	// Print the request data
	color.HiBlue("\nRequest %d data:\n", id)
	var buffer = make([]byte, 1024)
	for {
		n, err := r.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Println("Print: Error reading request data:", err)
			}
			return
		}
		color.Cyan(string(buffer[:n]))
	}

	if ready != nil {
		ready <- struct{}{}
	}
}

/*
Debug function to print the response data to the console
*/
func printResponse(r io.Reader, response *http.Response, id uint64, ready chan struct{}) {
	// Print the response data
	color.HiBlue("\nResponse %d data:\n", id)

	//Print status line
	color.HiGreen("HTTP/1.1 " + response.Status + "\r\n")
	//Print headers
	for k, v := range response.Header {
		color.HiGreen(k + ": " + strings.Join(v, ", ") + "\r\n")
	}
	color.Green("\r\n")

	var buffer = make([]byte, 2048)
	for {
		n, err :=
			r.Read(buffer)
		if err != nil {
			if err == io.EOF || err == io.ErrClosedPipe {
				break
			} else {
				log.Println("Print: Error reading response data:", err)
			}
			return
		}
		color.Green(string(buffer[:n]))
	}

	if ready != nil {
		ready <- struct{}{}
	}
}

func handleHttpConn(conn net.Conn, targetAddr net.Addr) {

	defer conn.Close()
	// //Create mock http response:
	// response := "HTTP/1.1 200 OK\r\n" +
	// 	"Content-Type: text/html\r\n" +
	// 	"Content-Length: 11\r\n" +
	// 	"\r\n" +
	// 	"Hello World" +
	// 	"\r\n"

	// Increment the counter for each http request
	counter++

	ready := make(chan struct{})
	printRequestReady := make(chan struct{})

	pr, pw := io.Pipe()
	printTee := io.TeeReader(conn, pw)

	go printRequest(printTee, counter, printRequestReady)

	go func() {
		defer pr.Close()
		defer pw.Close()

		// Read the request data
		request, err := parseHttpRequest(pr)
		if err != nil {
			log.Printf("HandleHttp: Failed to parse HTTP request: %v", err)
			return
		}
		//fixRequest(request)

		if shouldIntercept(request) {
			// Intercept the request
			// Send the response to the client

			log.Printf("Intercepting request %d", counter)

			err := intercept(*request, conn, targetAddr, counter)
			if err != nil {
				log.Printf("Failed to intercept request: %v", err)
				return
			}
		} else {
			// Request is not of interest, connect the client to the server directly

			log.Printf("Ignoring requuest %d", counter)
			connectClientToServerDirectly(conn, targetAddr, request)
		}

		ready <- struct{}{}
	}()

	<-ready
	<-printRequestReady

}

func parseHttpRequest(r io.Reader) (*http.Request, error) {
	// Parse the HTTP request data
	request, err := http.ReadRequest(bufio.NewReader(r))
	if err != nil {
		return nil, err
	}
	request.URL.Host = request.Host
	return request, nil
}

func shouldIntercept(request *http.Request) bool {
	for _, link := range interceptConfig.InterceptLinks {
		if !link.Intercept {
			continue
		}
		if strings.Contains(request.URL.String(), link.Url) {
			fmt.Printf("Intercepting request to %s\n", link.Url)
			return true
		}
	}
	//TODO: change logic here after fixing the intercept config
	return false
}

func intercept(request http.Request, conn net.Conn, targetAddr net.Addr, counter uint64) error {

	targetConn, err := net.DialTimeout("tcp", targetAddr.String(), 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to original destination: %w", err)
	}

	request.Write(targetConn)

	resChan := make(chan *http.Response)

	go func() {
		res, err := http.ReadResponse(bufio.NewReader(targetConn), &request)
		if err != nil {
			log.Printf("Failed to read response from target: %v", err)
			resChan <- nil
			return
		}
		resChan <- res
	}()

	go func() {
		defer targetConn.Close()
		defer conn.Close()
		res := <-resChan
		if res != nil {
			// Print the response data and forward it to the client

			// Create a multiwriter to write the response to the client and to the console
			consolePipeReader, consolePipeWriter := io.Pipe()
			defer consolePipeReader.Close()
			defer consolePipeWriter.Close()

			multiWriter := io.MultiWriter(conn, consolePipeWriter)

			go printResponse(consolePipeReader, res, counter, nil)

			// Write the headers first
			err := res.Write(multiWriter)
			if err != nil {
				log.Printf("Failed to write response headers to client: %v", err)
				return
			}
		}
	}()

	return nil
}

/* Connects the client to the server directly */
func connectClientToServerDirectly(conn net.Conn, targetAddr net.Addr, request *http.Request) {
	targetConn, err := net.DialTimeout("tcp", targetAddr.String(), 5*time.Second)
	if err != nil {
		log.Printf("HandleHttp: Failed to connect to original destination: %v", err)
		return
	}

	// Close the connection to the target server when the function returns
	defer targetConn.Close()
	defer conn.Close()

	// Forward the processed request to the target
	// The following code creates two data transfer channels:
	// - From the client to the target server (handled by a separate goroutine).
	// - From the target server to the client (handled by the main goroutine).
	ready := make(chan struct{})
	go func() {
		err := request.Write(targetConn)
		if err != nil {
			log.Printf("Failed to forward connection to target: %v", err)
			return
		}
		ready <- struct{}{}
	}()

	_, err = io.Copy(conn, targetConn)
	if err != nil {
		log.Printf("Failed to forward connection from target: %v", err)
		return
	}
	<-ready
}

func NewHttpPrintingProxy() *HttpPrintingProxy {
	initConfig()
	return &HttpPrintingProxy{}
}
