package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
)

type RequestHandler interface {
	AnalyzeRequest(data []byte) (shouldForward bool, err error)
}

type HttpRequestHandler struct {
}

func (h *HttpRequestHandler) AnalyzeRequest(data []byte) (bool, error) {
	if isHTTP(data) {
		request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(data)))

		if err != nil {
			return true, err
		}

		if request.URL.Host == "www.example.com" {
			requestDump, err := httputil.DumpRequestOut(request, true)
			if err != nil {
				return true, err
			}
			fmt.Printf("Request dump:\n%s\n", string(requestDump))
		}
	}

	return true, nil
}

func isHTTP(data []byte) bool {
	methods := []string{"GET", "POST", "HEAD", "PUT", "DELETE", "OPTIONS", "TRACE", "CONNECT"}
	for _, method := range methods {
		if bytes.HasPrefix(data, []byte(method+" ")) {
			return true
		}
	}
	return false
}
