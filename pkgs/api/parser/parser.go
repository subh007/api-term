package parser

import (
	"log"
	"net/url"

	"org.subh/api-term/pkgs/api/model"

	"github.com/getkin/kin-openapi/openapi3"
)

// ParseOpenAPI loads OpenAPI specs from multiple files and URLs
func ParseOpenAPI(filePaths []string, urls []string) []*model.Endpoint {
	loader := openapi3.NewLoader()
	var endpoints []*model.Endpoint

	// Helper to process a doc
	processDoc := func(doc *openapi3.T) {
		if err := doc.Validate(loader.Context); err != nil {
			log.Printf("Validation warning: %v", err)
		}

		for path, pathItem := range doc.Paths.Map() {
			for method, op := range map[string]*openapi3.Operation{
				"GET":    pathItem.Get,
				"POST":   pathItem.Post,
				"DELETE": pathItem.Delete,
				"PUT":    pathItem.Put,
				"PATCH":  pathItem.Patch,
			} {
				if op != nil {
					var params []*model.Parameter
					for _, paramRef := range op.Parameters {
						if paramRef.Value != nil {
							params = append(params, &model.Parameter{
								Name:     paramRef.Value.Name,
								In:       paramRef.Value.In,
								Required: paramRef.Value.Required,
							})
						}
					}
					endpoints = append(endpoints, &model.Endpoint{
						Method:     method,
						Path:       path,
						Parameters: params,
					})
				}
			}
		}
	}

	for _, filePath := range filePaths {
		if filePath == "" {
			continue
		}
		doc, err := loader.LoadFromFile(filePath)
		if err != nil {
			log.Printf("Failed to load file %s: %v", filePath, err)
			continue
		}
		processDoc(doc)
	}

	for _, u := range urls {
		if u == "" {
			continue
		}
		parsedURL, err := url.Parse(u)
		if err != nil {
			log.Printf("Invalid URL %s: %v", u, err)
			continue
		}
		doc, err := loader.LoadFromURI(parsedURL)
		if err != nil {
			log.Printf("Failed to load URL %s: %v", u, err)
			continue
		}
		processDoc(doc)
	}

	return endpoints
}
