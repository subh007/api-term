package main

import (
	"log"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"org.subh/api-term/pkgs/api/client"
	"org.subh/api-term/pkgs/api/parser"
	"org.subh/api-term/pkgs/config"
)

func main() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	// Load OpenAPI endpoints
	endpoints := parser.ParseOpenAPI(config.OpenAPIFile)

	// --- Setup UI ---
	list := widgets.NewList()
	list.Title = "API Endpoints (j/k to scroll, ENTER to select)"
	for _, ep := range endpoints {
		list.Rows = append(list.Rows, ep.Method+" "+ep.Path)
	}
	list.SelectedRow = 0
	list.TextStyle = ui.NewStyle(ui.ColorYellow)
	list.WrapText = false

	output := widgets.NewParagraph()
	output.Title = "Response"
	output.Text = "Press ENTER to invoke endpoint"
	output.WrapText = true

	input := widgets.NewParagraph()
	input.Title = "Query Parameters (param=value)"
	input.Text = ""
	input.BorderStyle.Fg = ui.ColorCyan

	termWidth, termHeight := ui.TerminalDimensions()
	list.SetRect(0, 0, termWidth, termHeight/2)
	output.SetRect(0, termHeight/2, termWidth, termHeight-3)
	input.SetRect(0, termHeight-3, termWidth, termHeight)

	ui.Render(list, output, input)

	// --- Event loop ---
	uiEvents := ui.PollEvents()
	// selected := 0
	userInput := ""
	inputMode := false

	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		case "j", "<Down>":
			if !inputMode && list.SelectedRow < len(list.Rows)-1 {
				list.SelectedRow++
			}
		case "k", "<Up>":
			if !inputMode && list.SelectedRow > 0 {
				list.SelectedRow--
			}
		case "<Enter>":
			if inputMode {
				inputMode = false
			} else {
				ep := endpoints[list.SelectedRow]
				inputValues := map[string]string{}
				if userInput != "" {
					// Simple query param parsing key=value&key2=value2
					pairs := strings.Split(userInput, "&")
					for _, p := range pairs {
						kv := strings.SplitN(p, "=", 2)
						if len(kv) == 2 {
							inputValues[kv[0]] = kv[1]
						}
					}
				}
				resp := client.InvokeEndpoint(config.BaseURL, ep, inputValues)
				output.Text = resp
				userInput = ""
				input.Text = ""
			}
		case "<Backspace>":
			if inputMode && len(userInput) > 0 {
				userInput = userInput[:len(userInput)-1]
				input.Text = userInput
			}
		case "i":
			inputMode = true
		default:
			if inputMode && len(e.ID) == 1 {
				userInput += e.ID
				input.Text = userInput
			}
		}

		ui.Render(list, input, output)
	}
}
