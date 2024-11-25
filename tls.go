package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

// RequestHandler decides how to handle HTTP requests.
type RequestHandler func(req *http.Request) (*http.Response, bool)

// ResponseHandler decides how to handle HTTP responses.
type ResponseHandler func(resp *http.Response) (*http.Response, bool)

// TLSConfig is a placeholder for managing TLS certificates.
var tlsConfig *tls.Config

func handleTLSConnection(conn net.Conn, reqHandler RequestHandler, respHandler ResponseHandler) {

	defer conn.Close()

	// Perform TLS handshake with client
	clientTLS := tls.Server(conn, tlsConfig)
	if err := clientTLS.Handshake(); err != nil {
		log.Printf("TLS handshake failed: %v", err)
		return
	}

	// Read and parse HTTP request
	req, err := http.ReadRequest(bufio.NewReader(clientTLS))
	if err != nil {
		log.Printf("Failed to read request: %v", err)
		return
	}

	// Call the request handler
	resp, forward := reqHandler(req)
	if !forward {
		// Respond locally if instructed
		if resp != nil {
			resp.Write(clientTLS)
		}
		return
	}

	// Forward request to destination server
	targetConn, err := tls.Dial("tcp", req.Host, tlsConfig)
	if err != nil {
		log.Printf("Failed to connect to target server: %v", err)
		return
	}
	defer targetConn.Close()

	if err := req.Write(targetConn); err != nil {
		log.Printf("Failed to forward request: %v", err)
		return
	}

	// Read server's response
	serverResp, err := http.ReadResponse(bufio.NewReader(targetConn), req)
	if err != nil {
		log.Printf("Failed to read server response: %v", err)
		return
	}

	// Call the response handler
	modifiedResp, forward := respHandler(serverResp)
	if forward && modifiedResp != nil {
		// Forward response to client
		modifiedResp.Write(clientTLS)
	} else if modifiedResp != nil {
		// Respond locally to client
		modifiedResp.Write(clientTLS)
	}
}

func start_tls_proxy(address string) {
	// Load TLS configuration (CA certificate, dynamic cert generation)
	tlsConfig = loadTLSConfig()

	// Start the proxy server
	listener, err := tls.Listen("tcp", address, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}
	defer listener.Close()

	log.Printf("Proxy server listening on %s", listener.Addr())
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleTLSConnection(conn, exampleRequestHandler, exampleResponseHandler)
	}
}

// exampleRequestHandler processes HTTP requests.
func exampleRequestHandler(req *http.Request) (*http.Response, bool) {
	fmt.Printf("Request: %s %s\n", req.Method, req.URL)
	// Example: Block requests to a specific host
	if req.Host == "blocked.example.com" {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader("Blocked by proxy")),
		}, false
	}
	return nil, true
}

// exampleResponseHandler processes HTTP responses.
func exampleResponseHandler(resp *http.Response) (*http.Response, bool) {
	fmt.Printf("Response: %d %s\n", resp.StatusCode, resp.Status)
	return resp, true
}

// loadTLSConfig loads TLS configuration.
func loadTLSConfig() *tls.Config {
	// Example: Load CA certificate
	caCert, err := os.ReadFile("ca-cert.crt")
	if err != nil {
		log.Fatalf("Failed to load CA certificate: %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Example: Generate a self-signed certificate
	cert, err := tls.LoadX509KeyPair("ca-cert.crt", "ca-key.pem")
	if err != nil {
		log.Fatalf("Failed to load certificate: %v", err)
	}

	return &tls.Config{
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
	}
}
