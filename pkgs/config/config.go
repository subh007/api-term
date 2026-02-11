package config

type Config struct {
	BaseURL     string
	OpenAPIFile string
	OpenAPIURLs []string
}

var DefaultBaseURL = "http://localhost:8080"
var DefaultOpenAPIFile = "assets/api.yaml"

func New(openAPIFile string, openAPIURLs []string) *Config {
	return &Config{
		BaseURL:     DefaultBaseURL,
		OpenAPIFile: openAPIFile,
		OpenAPIURLs: openAPIURLs,
	}
}
