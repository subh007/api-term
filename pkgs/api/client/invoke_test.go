package client

import (
	"strings"
	"testing"

	"org.subh/api-term/pkgs/api/model"
)

func TestInvokeEndpoint_MissingPathParam(t *testing.T) {
	ep := &model.Endpoint{
		Method: "GET",
		Path:   "/test/{id}",
		Parameters: []*model.Parameter{
			{
				Name:     "id",
				In:       "path",
				Required: true,
			},
		},
	}

	inputValues := map[string]string{}
	_, _, err := InvokeEndpoint("http://localhost", ep, inputValues, nil)

	if err == nil {
		t.Fatal("Expected error for missing path param, got nil")
	}

	expectedError := "Missing path param: id"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}
}

func TestInvokeEndpoint_MissingQueryParam(t *testing.T) {
	ep := &model.Endpoint{
		Method: "GET",
		Path:   "/test",
		Parameters: []*model.Parameter{
			{
				Name:     "q",
				In:       "query",
				Required: true,
			},
		},
	}

	inputValues := map[string]string{}
	_, _, err := InvokeEndpoint("http://localhost", ep, inputValues, nil)

	if err == nil {
		t.Fatal("Expected error for missing query param, got nil")
	}

	expectedError := "Missing query param: q"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}
}
