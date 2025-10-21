package model

type Parameter struct {
	Name     string
	In       string // "path" or "query"
	Required bool
}

type Endpoint struct {
	Method     string
	Path       string
	Parameters []*Parameter
}
