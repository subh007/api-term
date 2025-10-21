package parser

import (
	"log"

	"org.subh/api-term/pkgs/api/model"

	"github.com/getkin/kin-openapi/openapi3"
)

// ParseOpenAPI loads a YAML and returns Endpoint list
func ParseOpenAPI(filePath string) []*model.Endpoint {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(filePath)
	if err != nil {
		log.Fatal(err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		log.Fatal(err)
	}

	var endpoints []*model.Endpoint
	for path, pathItem := range doc.Paths.Map() {
		for method, op := range map[string]*openapi3.Operation{
			"GET":    pathItem.Get,
			"POST":   pathItem.Post,
			"DELETE": pathItem.Delete,
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

	return endpoints
}
