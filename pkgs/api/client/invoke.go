package client

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"org.subh/api-term/pkgs/api/model"
)

func InvokeEndpoint(baseURL string, ep *model.Endpoint, inputValues map[string]string) (string, int, error) {
	finalPath := ep.Path

	// Fill path parameters
	for _, param := range ep.Parameters {
		if param.In == "path" {
			val, ok := inputValues[param.Name]
			if !ok {
				return "", 0, fmt.Errorf("Missing path param: %s", param.Name)
			}
			finalPath = strings.Replace(finalPath, "{"+param.Name+"}", val, 1)
		}
	}

	// Append query params
	var queryParts []string
	for _, param := range ep.Parameters {
		if param.In == "query" {
			if val, ok := inputValues[param.Name]; ok {
				queryParts = append(queryParts, fmt.Sprintf("%s=%s", param.Name, val))
			}
		}
	}
	if len(queryParts) > 0 {
		finalPath += "?" + strings.Join(queryParts, "&")
	}

	url := baseURL + finalPath

	resp, err := http.Get(url)
	if err != nil {
		return "", 0, fmt.Errorf("Error: %v", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return string(body), resp.StatusCode, nil
}
