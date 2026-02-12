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

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	// Load OpenAPI endpoints
	// For backward compatibility, treat fileFlag as a single file in a slice
	files := []string{cfg.OpenAPIFile}
	endpoints := parser.ParseOpenAPI(files, cfg.OpenAPIURLs)

	// --- Setup UI ---
	list := widgets.NewList()
	list.Title = "API Endpoints (j/k to scroll, ENTER to select)"
	for _, ep := range endpoints {
		list.Rows = append(list.Rows, formatEndpointRow(ep))
	}
	list.SelectedRow = 0
	list.TextStyle = ui.NewStyle(ui.ColorYellow)
	list.WrapText = false

	output := widgets.NewList()
	output.Title = "Response (Tab/r to focus, j/k to scroll)"
	output.Rows = []string{"Press ENTER to invoke endpoint"}
	output.WrapText = true
	output.TextStyle = ui.NewStyle(ui.ColorWhite)
	output.SelectedRowStyle = ui.NewStyle(ui.ColorWhite, ui.ColorBlack) // default to no highlight until focused

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

	termWidth, termHeight := ui.TerminalDimensions()
	list.SetRect(0, 0, termWidth, termHeight/2)
	output.SetRect(0, termHeight/2, termWidth, termHeight-9)
	baseURLWidget.SetRect(0, termHeight-9, termWidth, termHeight-6)
	headersWidget.SetRect(0, termHeight-6, termWidth, termHeight-3)
	input.SetRect(0, termHeight-3, termWidth, termHeight)

	// Help Overlay
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
	help.SetRect(termWidth/4, termHeight/4, 3*termWidth/4, 3*termHeight/4)
	help.BorderStyle.Fg = ui.ColorYellow

	ui.Render(list, output, baseURLWidget, headersWidget, input)

	// --- Event loop ---
	uiEvents := ui.PollEvents()
	// selected := 0
	baseURL := cfg.BaseURL
	queryInput := ""
	headerInput := ""
	editBuffer := ""
	inputMode := false
	editTarget := ""
	showHelp := false

	// focusMode: "list" (endpoints) or "output" (response)
	focusMode := "list"

	for {
		e := <-uiEvents
		if showHelp {
			// ...
			switch e.ID {
			case "?", "h", "<Escape>":
				showHelp = false
			}
		} else if inputMode {
			switch e.ID {
			case "<Enter>":
				inputMode = false
				switch editTarget {
				case "params":
					queryInput = editBuffer
					input.Text = queryInput
					input.BorderStyle.Fg = ui.ColorCyan
				case "baseurl":
					trimmed := strings.TrimSpace(editBuffer)
					if trimmed != "" {
						baseURL = trimmed
						baseURLWidget.Text = baseURL
					} else {
						baseURLWidget.Text = baseURL
					}
					baseURLWidget.BorderStyle.Fg = ui.ColorMagenta
				case "headers":
					headerInput = strings.TrimSpace(editBuffer)
					headersWidget.Text = headerInput
					headersWidget.BorderStyle.Fg = ui.ColorBlue
				}
				editTarget = ""
			case "<Backspace>":
				if len(editBuffer) > 0 {
					editBuffer = editBuffer[:len(editBuffer)-1]
					if editTarget == "baseurl" {
						baseURLWidget.Text = editBuffer
					} else if editTarget == "headers" {
						headersWidget.Text = editBuffer
					} else {
						input.Text = editBuffer
					}
				}
			case "<C-c>":
				return
			default:
				if len(e.ID) == 1 {
					editBuffer += e.ID
					if editTarget == "baseurl" {
						baseURLWidget.Text = editBuffer
					} else if editTarget == "headers" {
						headersWidget.Text = editBuffer
					} else {
						input.Text = editBuffer
					}
				}
			}
		} else {
			switch e.ID {
			case "q", "<C-c>":
				return
			case "<Tab>", "r":
				if focusMode == "list" {
					focusMode = "output"
					list.TitleStyle = ui.NewStyle(ui.ColorWhite)
					list.BorderStyle.Fg = ui.ColorWhite
					output.TitleStyle = ui.NewStyle(ui.ColorYellow)
					output.BorderStyle.Fg = ui.ColorYellow
				} else {
					focusMode = "list"
					list.TitleStyle = ui.NewStyle(ui.ColorYellow)
					list.BorderStyle.Fg = ui.ColorYellow
					output.TitleStyle = ui.NewStyle(ui.ColorWhite)
					output.BorderStyle.Fg = ui.ColorWhite
				}
			case "j", "<Down>":
				if focusMode == "list" {
					if list.SelectedRow < len(list.Rows)-1 {
						list.SelectedRow++
					}
				} else {
					if output.SelectedRow < len(output.Rows)-1 {
						output.SelectedRow++
					}
				}
			case "k", "<Up>":
				if focusMode == "list" {
					if list.SelectedRow > 0 {
						list.SelectedRow--
					}
				} else {
					if output.SelectedRow > 0 {
						output.SelectedRow--
					}
				}
			case "<Enter>":
				if focusMode == "output" {
					continue
				}
				ep := endpoints[list.SelectedRow]
				inputValues := map[string]string{}
				// Initialize with global query params
				for k, v := range cfg.GlobalQueryParams {
					inputValues[k] = v
				}
				headerValues := map[string]string{}

				// Check for shorthand input (single required path param, no '=')
				var requiredPathParams []string
				var requiredQueryParams []string
				for _, p := range ep.Parameters {
					if p.Required && p.In == "path" {
						requiredPathParams = append(requiredPathParams, p.Name)
					} else if p.Required && p.In == "query" {
						requiredQueryParams = append(requiredQueryParams, p.Name)
					}
				}

				if queryInput != "" && !strings.Contains(queryInput, "=") && !strings.Contains(queryInput, "&") {
					if len(requiredPathParams) == 1 && len(requiredQueryParams) == 0 {
						inputValues[requiredPathParams[0]] = queryInput
					} else if len(requiredPathParams) == 0 && len(requiredQueryParams) == 1 {
						inputValues[requiredQueryParams[0]] = queryInput
					}
				} else if queryInput != "" {
					// Simple query param parsing key=value&key2=value2
					pairs := strings.Split(queryInput, "&")
					for _, p := range pairs {
						kv := strings.SplitN(p, "=", 2)
						if len(kv) == 2 {
							inputValues[kv[0]] = kv[1]
						}
					}
				}
				if headerInput != "" {
					pairs := strings.FieldsFunc(headerInput, func(r rune) bool {
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
				resp, statusCode, err := client.InvokeEndpoint(baseURL, ep, inputValues, headerValues)
				statusColor := "green" // default success
				output.BorderStyle.Fg = ui.ColorGreen

				if statusCode >= 400 {
					statusColor = "red"
					output.BorderStyle.Fg = ui.ColorRed
				}

				if err != nil {
					output.Rows = []string{fmt.Sprintf("Error: %s", err.Error()), fmt.Sprintf("[Status: %d](fg:%s)", statusCode, statusColor)}
					output.BorderStyle.Fg = ui.ColorRed
				} else {
					formattedResp := tryFormatJSON(resp)
					headerLine := fmt.Sprintf("[Status: %d](fg:%s)", statusCode, statusColor)
					output.Rows = append([]string{headerLine, ""}, splitLines(formattedResp)...)
					output.BorderStyle.Fg = ui.ColorGreen
				}
				output.SelectedRow = 0
				queryInput = ""
				input.Text = ""
			case "i":
				inputMode = true
				editTarget = "params"
				editBuffer = queryInput
				input.Text = editBuffer
				input.BorderStyle.Fg = ui.ColorYellow
				output.BorderStyle.Fg = ui.ColorWhite
			case "b":
				inputMode = true
				editTarget = "baseurl"
				editBuffer = baseURL
				baseURLWidget.Text = editBuffer
				baseURLWidget.BorderStyle.Fg = ui.ColorYellow
				output.BorderStyle.Fg = ui.ColorWhite
			case "H":
				inputMode = true
				editTarget = "headers"
				editBuffer = headerInput
				headersWidget.Text = editBuffer
				headersWidget.BorderStyle.Fg = ui.ColorYellow
				output.BorderStyle.Fg = ui.ColorWhite
			case "?", "h":
				showHelp = true
			}
		}

		// Update input title based on selected endpoint
		currEp := endpoints[list.SelectedRow]
		var requiredParams []string
		for _, p := range currEp.Parameters {
			if p.Required {
				requiredParams = append(requiredParams, p.Name+" ("+p.In+")")
			}
		}
		if len(requiredParams) > 0 {
			input.Title = fmt.Sprintf("Query Parameters (Required: %s) - Press 'i' to edit", strings.Join(requiredParams, ", "))
		} else {
			input.Title = "Query Parameters (param=value) - Press 'i' to edit"
		}

		if showHelp {
			ui.Render(help)
		} else {
			ui.Render(list, output, baseURLWidget, headersWidget, input)
		}
	}
}
