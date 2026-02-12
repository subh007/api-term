package parser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseOpenAPI_URL(t *testing.T) {
	// Mock server serving OpenAPI spec
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `openapi: 3.0.0
info:
  title: Sample API
  version: 0.1.9
paths:
  /users:
    get:
      summary: Returns a list of users.
      responses:
        '200':
          description: A JSON array of user names
`)
	}))
	defer ts.Close()

	endpoints := ParseOpenAPI(nil, []string{ts.URL})

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	// Iterate to find the endpoint (map iteration order is random, but here we only have one)
	ep := endpoints[0]
	if ep.Path != "/users" {
		t.Errorf("expected path /users, got %s", ep.Path)
	}
	if ep.Method != "GET" {
		t.Errorf("expected method GET, got %s", ep.Method)
	}
}
