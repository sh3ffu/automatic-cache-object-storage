package proxy

import (
	"net/http"
	"testing"
)

func TestShouldIntercept(t *testing.T) {
	// Test if the shouldIntercept function returns true when the URL is in the intercept list
	// and false when it is not
	interceptConfig = InterceptConfig{
		ConfigName: "default",
		InterceptLinks: []InterceptLink{
			{
				Url:       "http://example.com",
				Intercept: true,
			},
			{
				Url:       "http://example.org",
				Intercept: false,
			},
		},
	}

	testRequest1, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Errorf("Error creating test request: %v", err)
		return
	}

	testRequest2, err := http.NewRequest("GET", "http://example.org", nil)
	if err != nil {
		t.Errorf("Error creating test request: %v", err)
		return
	}

	testRequest3, err := http.NewRequest("GET", "http://example.net", nil)
	if err != nil {
		t.Errorf("Error creating test request: %v", err)
		return
	}

	if shouldIntercept(testRequest1) != true {
		t.Errorf("Expected true, got false")
	}

	if shouldIntercept(testRequest2) != false {
		t.Errorf("Expected false, got true")
	}

	if shouldIntercept(testRequest3) != false {
		t.Errorf("Expected false, got true")
	}
}
