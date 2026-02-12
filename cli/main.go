package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"org.subh/api-term/pkgs/api/client"
	"org.subh/api-term/pkgs/api/model"
	"org.subh/api-term/pkgs/api/parser"
	"org.subh/api-term/pkgs/config"
	"org.subh/api-term/pkgs/tui"
)

// stringSlice implements flag.Value for multiple flags
type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprint(*s)
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func formatEndpointRow(ep *model.Endpoint) string {
	var queryParams []string
	for _, p := range ep.Parameters {
		if p.In == "query" {
			queryParams = append(queryParams, p.Name)
		}
	}
	if len(queryParams) == 0 {
		return ep.Method + " " + ep.Path
	}
	return fmt.Sprintf("%s %s?%s", ep.Method, ep.Path, strings.Join(queryParams, "&"))
}

type MainHandler struct {
	Endpoints         []*model.Endpoint
	GlobalQueryParams map[string]string
	Config            *config.Config

	// UI Widgets
	List          *widgets.List
	Output        *widgets.List
	BaseURLWidget *widgets.Paragraph
	HeadersWidget *widgets.Paragraph
	Input         *widgets.Paragraph
	Help          *widgets.Paragraph

	// State
	FocusMode    string
	ShowHelp     bool
	InputMode    bool
	EditTarget   string
	EditBuffer   string
	QueryInput   string
	HeaderInput  string
	BaseURL      string
	InputValues  map[string]string
	HeaderValues map[string]string
}

func NewMainHandler(cfg *config.Config, endpoints []*model.Endpoint) *MainHandler {
	list := widgets.NewList()
	list.Title = "API Endpoints (j/k to scroll, ENTER to select)"
	for _, ep := range endpoints {
		list.Rows = append(list.Rows, formatEndpointRow(ep))
	}
	list.SelectedRow = 0
	list.TextStyle = ui.NewStyle(ui.ColorYellow)
	list.WrapText = false
	list.TitleStyle = ui.NewStyle(ui.ColorYellow)
	list.BorderStyle.Fg = ui.ColorYellow

	output := widgets.NewList()
	output.Title = "Response (Tab/r to focus, j/k to scroll)"
	output.Rows = []string{"Press ENTER to invoke endpoint"}
	output.WrapText = true
	output.TextStyle = ui.NewStyle(ui.ColorWhite)
	output.SelectedRowStyle = ui.NewStyle(ui.ColorWhite, ui.ColorBlack)
	output.BorderStyle.Fg = ui.ColorWhite

	baseURLWidget := widgets.NewParagraph()
	baseURLWidget.Title = "Base URL (press 'b' to edit)"
	baseURLWidget.Text = cfg.BaseURL
	baseURLWidget.BorderStyle.Fg = ui.ColorMagenta

	headersWidget := widgets.NewParagraph()
	headersWidget.Title = "Headers (key:value&key2:value2) - Press 'H' to edit"
	headersWidget.Text = ""
	headersWidget.BorderStyle.Fg = ui.ColorBlue

	input := widgets.NewParagraph()
	input.Title = "Query Parameters (param=value) - Press 'i' to edit"
	input.Text = ""
	input.BorderStyle.Fg = ui.ColorCyan

	help := widgets.NewParagraph()
	help.Title = "Help"
	help.Text = `
	Navigation Keys:
	  Tab / r      Toggle Focus (Endpoints <-> Response)
	  j / <Down>   Scroll Down (Endpoints or Response)
	  k / <Up>     Scroll Up
	  Enter        Select Endpoint / Invoke
	  i            Focus Input
	  b            Edit Base URL
	  H            Edit Headers
	  ? / h        Toggle Help
	  q / <C-c>    Quit
	`
	help.BorderStyle.Fg = ui.ColorYellow

	h := &MainHandler{
		Endpoints:         endpoints,
		GlobalQueryParams: cfg.GlobalQueryParams,
		Config:            cfg,
		List:              list,
		Output:            output,
		BaseURLWidget:     baseURLWidget,
		HeadersWidget:     headersWidget,
		Input:             input,
		Help:              help,
		FocusMode:         "list",
		BaseURL:           cfg.BaseURL,
		InputValues:       make(map[string]string),
		HeaderValues:      make(map[string]string),
	}

	return h
}

func (h *MainHandler) Init(termWidth, termHeight int) {
	h.Resize(termWidth, termHeight)
}

func (h *MainHandler) Resize(termWidth, termHeight int) {
	h.List.SetRect(0, 0, termWidth, termHeight/2)
	h.Output.SetRect(0, termHeight/2, termWidth, termHeight-9)
	h.BaseURLWidget.SetRect(0, termHeight-9, termWidth, termHeight-6)
	h.HeadersWidget.SetRect(0, termHeight-6, termWidth, termHeight-3)
	h.Input.SetRect(0, termHeight-3, termWidth, termHeight)
	h.Help.SetRect(termWidth/4, termHeight/4, 3*termWidth/4, 3*termHeight/4)
}

func (h *MainHandler) Render() {
	if h.ShowHelp {
		ui.Render(h.Help)
	} else {
		// Update input title based on selected endpoint
		currEp := h.Endpoints[h.List.SelectedRow]
		var requiredParams []string
		for _, p := range currEp.Parameters {
			if p.Required {
				requiredParams = append(requiredParams, p.Name+" ("+p.In+")")
			}
		}
		if len(requiredParams) > 0 {
			h.Input.Title = fmt.Sprintf("Query Parameters (Required: %s) - Press 'i' to edit", strings.Join(requiredParams, ", "))
		} else {
			h.Input.Title = "Query Parameters (param=value) - Press 'i' to edit"
		}

		ui.Render(h.List, h.Output, h.BaseURLWidget, h.HeadersWidget, h.Input)
	}
}

func (h *MainHandler) HandleEvent(e tui.Event) bool {
	if e.Type == ui.ResizeEvent {
		payload := e.Payload.(ui.Resize)
		h.Resize(payload.Width, payload.Height)
		return false
	}

	if h.ShowHelp {
		switch e.ID {
		case "?", "h", "<Escape>":
			h.ShowHelp = false
		}
		return false
	}

	if h.InputMode {
		switch e.ID {
		case "<Enter>":
			h.InputMode = false
			switch h.EditTarget {
			case "params":
				h.QueryInput = h.EditBuffer
				h.Input.Text = h.QueryInput
				h.Input.BorderStyle.Fg = ui.ColorCyan
			case "baseurl":
				trimmed := strings.TrimSpace(h.EditBuffer)
				if trimmed != "" {
					h.BaseURL = trimmed
					h.BaseURLWidget.Text = h.BaseURL
				} else {
					h.BaseURLWidget.Text = h.BaseURL
				}
				h.BaseURLWidget.BorderStyle.Fg = ui.ColorMagenta
			case "headers":
				h.HeaderInput = strings.TrimSpace(h.EditBuffer)
				h.HeadersWidget.Text = h.HeaderInput
				h.HeadersWidget.BorderStyle.Fg = ui.ColorBlue
			}
			h.EditTarget = ""
		case "<Backspace>":
			if len(h.EditBuffer) > 0 {
				h.EditBuffer = h.EditBuffer[:len(h.EditBuffer)-1]
				if h.EditTarget == "baseurl" {
					h.BaseURLWidget.Text = h.EditBuffer
				} else if h.EditTarget == "headers" {
					h.HeadersWidget.Text = h.EditBuffer
				} else {
					h.Input.Text = h.EditBuffer
				}
			}
		case "<C-c>":
			return true
		default:
			if len(e.ID) == 1 {
				h.EditBuffer += e.ID
				if h.EditTarget == "baseurl" {
					h.BaseURLWidget.Text = h.EditBuffer
				} else if h.EditTarget == "headers" {
					h.HeadersWidget.Text = h.EditBuffer
				} else {
					h.Input.Text = h.EditBuffer
				}
			}
		}
		return false
	}

	switch e.ID {
	case "q", "<C-c>":
		return true
	case "<Tab>", "r":
		if h.FocusMode == "list" {
			h.FocusMode = "output"
			h.List.TitleStyle = ui.NewStyle(ui.ColorWhite)
			h.List.BorderStyle.Fg = ui.ColorWhite
			h.Output.TitleStyle = ui.NewStyle(ui.ColorYellow)
			h.Output.BorderStyle.Fg = ui.ColorYellow
		} else {
			h.FocusMode = "list"
			h.List.TitleStyle = ui.NewStyle(ui.ColorYellow)
			h.List.BorderStyle.Fg = ui.ColorYellow
			h.Output.TitleStyle = ui.NewStyle(ui.ColorWhite)
			h.Output.BorderStyle.Fg = ui.ColorWhite
		}
	case "j", "<Down>":
		if h.FocusMode == "list" {
			if h.List.SelectedRow < len(h.List.Rows)-1 {
				h.List.SelectedRow++
			}
		} else {
			if h.Output.SelectedRow < len(h.Output.Rows)-1 {
				h.Output.SelectedRow++
			}
		}
	case "k", "<Up>":
		if h.FocusMode == "list" {
			if h.List.SelectedRow > 0 {
				h.List.SelectedRow--
			}
		} else {
			if h.Output.SelectedRow > 0 {
				h.Output.SelectedRow--
			}
		}
	case "<Enter>":
		if h.FocusMode == "output" {
			return false
		}
		ep := h.Endpoints[h.List.SelectedRow]
		inputValues := map[string]string{}
		// Initialize with global query params
		for k, v := range h.GlobalQueryParams {
			inputValues[k] = v
		}
		headerValues := map[string]string{}

		// Check for shorthand input
		var requiredPathParams []string
		var requiredQueryParams []string
		for _, p := range ep.Parameters {
			if p.Required && p.In == "path" {
				requiredPathParams = append(requiredPathParams, p.Name)
			} else if p.Required && p.In == "query" {
				requiredQueryParams = append(requiredQueryParams, p.Name)
			}
		}

		if h.QueryInput != "" && !strings.Contains(h.QueryInput, "=") && !strings.Contains(h.QueryInput, "&") {
			if len(requiredPathParams) == 1 && len(requiredQueryParams) == 0 {
				inputValues[requiredPathParams[0]] = h.QueryInput
			} else if len(requiredPathParams) == 0 && len(requiredQueryParams) == 1 {
				inputValues[requiredQueryParams[0]] = h.QueryInput
			}
		} else if h.QueryInput != "" {
			// Simple query param parsing key=value&key2=value2
			pairs := strings.Split(h.QueryInput, "&")
			for _, p := range pairs {
				kv := strings.SplitN(p, "=", 2)
				if len(kv) == 2 {
					inputValues[kv[0]] = kv[1]
				}
			}
		}
		if h.HeaderInput != "" {
			pairs := strings.FieldsFunc(h.HeaderInput, func(r rune) bool {
				return r == '&' || r == ';'
			})
			for _, p := range pairs {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				kv := strings.SplitN(p, ":", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					val := strings.TrimSpace(kv[1])
					if key != "" {
						headerValues[key] = val
					}
					continue
				}
				kv = strings.SplitN(p, "=", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					val := strings.TrimSpace(kv[1])
					if key != "" {
						headerValues[key] = val
					}
				}
			}
		}
		resp, statusCode, err := client.InvokeEndpoint(h.BaseURL, ep, inputValues, headerValues)
		statusColor := "green" // default success
		h.Output.BorderStyle.Fg = ui.ColorGreen

		if statusCode >= 400 {
			statusColor = "red"
			h.Output.BorderStyle.Fg = ui.ColorRed
		}

		if err != nil {
			h.Output.Rows = []string{fmt.Sprintf("Error: %s", err.Error()), fmt.Sprintf("[Status: %d](fg:%s)", statusCode, statusColor)}
			h.Output.BorderStyle.Fg = ui.ColorRed
		} else {
			formattedResp := tryFormatJSON(resp)
			headerLine := fmt.Sprintf("[Status: %d](fg:%s)", statusCode, statusColor)
			h.Output.Rows = append([]string{headerLine, ""}, splitLines(formattedResp)...)
			h.Output.BorderStyle.Fg = ui.ColorGreen
		}
		h.Output.SelectedRow = 0
		h.QueryInput = ""
		h.Input.Text = ""
	case "i":
		h.InputMode = true
		h.EditTarget = "params"
		h.EditBuffer = h.QueryInput
		h.Input.Text = h.EditBuffer
		h.Input.BorderStyle.Fg = ui.ColorYellow
		h.Output.BorderStyle.Fg = ui.ColorWhite
	case "b":
		h.InputMode = true
		h.EditTarget = "baseurl"
		h.EditBuffer = h.BaseURL
		h.BaseURLWidget.Text = h.EditBuffer
		h.BaseURLWidget.BorderStyle.Fg = ui.ColorYellow
		h.Output.BorderStyle.Fg = ui.ColorWhite
	case "H":
		h.InputMode = true
		h.EditTarget = "headers"
		h.EditBuffer = h.HeaderInput
		h.HeadersWidget.Text = h.EditBuffer
		h.HeadersWidget.BorderStyle.Fg = ui.ColorYellow
		h.Output.BorderStyle.Fg = ui.ColorWhite
	case "?", "h":
		h.ShowHelp = true
	}

	return false
}

func main() {
	// parse CLI flags
	fileFlag := flag.String("file", config.DefaultOpenAPIFile, "path to OpenAPI file")
	var urlFlags stringSlice
	flag.Var(&urlFlags, "url", "URL to OpenAPI spec (can be repeated)")
	var queryFlags stringSlice
	flag.Var(&queryFlags, "q", "Global query param key=value (can be repeated)")
	flag.Var(&queryFlags, "query", "Global query param key=value (can be repeated)")
	flag.Parse()

	globalQueryParams := make(map[string]string)
	for _, q := range queryFlags {
		parts := strings.SplitN(q, "=", 2)
		if len(parts) == 2 {
			globalQueryParams[parts[0]] = parts[1]
		}
	}

	cfg := config.New(*fileFlag, urlFlags, globalQueryParams)

	// Load OpenAPI endpoints
	// For backward compatibility, treat fileFlag as a single file in a slice
	files := []string{cfg.OpenAPIFile}
	endpoints := parser.ParseOpenAPI(files, cfg.OpenAPIURLs)

	handler := NewMainHandler(cfg, endpoints)
	app := tui.NewApp(handler)

	if err := app.Run(); err != nil {
		log.Fatalf("Action failed: %v", err)
	}
}
