package client

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"org.subh/api-term/pkgs/api/model"
)

func InvokeEndpoint(baseURL string, ep *model.Endpoint, inputValues map[string]string, headerValues map[string]string) (string, int, error) {
	finalPath := ep.Path

	usedParams := make(map[string]bool)

	// Fill path parameters
	for _, param := range ep.Parameters {
		if param.In == "path" {
			val, ok := inputValues[param.Name]
			if !ok {
				return "", 0, fmt.Errorf("Missing path param: %s", param.Name)
			}
			finalPath = strings.Replace(finalPath, "{"+param.Name+"}", val, 1)
			usedParams[param.Name] = true
		}
	}

	// Append query params
	var queryParts []string
	for _, param := range ep.Parameters {
		if param.In == "query" {
			val, ok := inputValues[param.Name]
			if !ok {
				if param.Required {
					return "", 0, fmt.Errorf("Missing query param: %s", param.Name)
				}
				continue
			}
			if param.Name != "" {
				queryParts = append(queryParts, fmt.Sprintf("%s=%s", url.QueryEscape(param.Name), url.QueryEscape(val)))
				usedParams[param.Name] = true
			}
		}
	}

	// Add any remaining input values as query params
	for k, v := range inputValues {
		if !usedParams[k] {
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v)))
		}
	}

	if len(queryParts) > 0 {
		finalPath += "?" + strings.Join(queryParts, "&")
	}

	url := baseURL + finalPath

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", 0, fmt.Errorf("Error: %v", err)
	}
	for k, v := range headerValues {
		if k != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("Error: %v", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return string(body), resp.StatusCode, nil
}
