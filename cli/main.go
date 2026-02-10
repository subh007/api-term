package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"org.subh/api-term/pkgs/api/client"
	"org.subh/api-term/pkgs/api/parser"
	"org.subh/api-term/pkgs/config"
)

func main() {
	// parse CLI flags
	fileFlag := flag.String("file", config.DefaultOpenAPIFile, "path to OpenAPI file")
	flag.Parse()

	cfg := config.New(*fileFlag)

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	// Load OpenAPI endpoints
	endpoints := parser.ParseOpenAPI(cfg.OpenAPIFile)

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
	input.Title = "Query Parameters (param=value) - Press 'i' to edit"
	input.Text = ""
	input.BorderStyle.Fg = ui.ColorCyan

	termWidth, termHeight := ui.TerminalDimensions()
	list.SetRect(0, 0, termWidth, termHeight/2)
	output.SetRect(0, termHeight/2, termWidth, termHeight-3)
	input.SetRect(0, termHeight-3, termWidth, termHeight)

	// Help Overlay
	help := widgets.NewParagraph()
	help.Title = "Help"
	help.Text = `
	Navigation Keys:
	  j / <Down>   Navigate Down
	  k / <Up>     Navigate Up
	  Enter        Select Endpoint / Invoke
	  i            Focus Input
	  ? / h        Toggle Help
	  q / <C-c>    Quit
	`
	help.SetRect(termWidth/4, termHeight/4, 3*termWidth/4, 3*termHeight/4)
	help.BorderStyle.Fg = ui.ColorYellow

	ui.Render(list, output, input)

	// --- Event loop ---
	uiEvents := ui.PollEvents()
	// selected := 0
	userInput := ""
	inputMode := false
	showHelp := false

	for {
		e := <-uiEvents
		if showHelp {
			switch e.ID {
			case "?", "h", "<Escape>":
				showHelp = false
			}
		} else if inputMode {
			switch e.ID {
			case "<Enter>":
				inputMode = false
			case "<Backspace>":
				if len(userInput) > 0 {
					userInput = userInput[:len(userInput)-1]
					input.Text = userInput
				}
			case "<C-c>":
				return
			default:
				if len(e.ID) == 1 {
					userInput += e.ID
					input.Text = userInput
				}
			}
		} else {
			switch e.ID {
			case "q", "<C-c>":
				return
			case "j", "<Down>":
				if list.SelectedRow < len(list.Rows)-1 {
					list.SelectedRow++
					output.BorderStyle.Fg = ui.ColorWhite
				}
			case "k", "<Up>":
				if list.SelectedRow > 0 {
					list.SelectedRow--
					output.BorderStyle.Fg = ui.ColorWhite
				}
			case "<Enter>":
				ep := endpoints[list.SelectedRow]
				inputValues := map[string]string{}

				// Check for shorthand input (single required path param, no '=')
				var requiredPathParams []string
				for _, p := range ep.Parameters {
					if p.Required && p.In == "path" {
						requiredPathParams = append(requiredPathParams, p.Name)
					}
				}

				if len(requiredPathParams) == 1 && !strings.Contains(userInput, "=") && userInput != "" {
					inputValues[requiredPathParams[0]] = userInput
				} else if userInput != "" {
					// Simple query param parsing key=value&key2=value2
					pairs := strings.Split(userInput, "&")
					for _, p := range pairs {
						kv := strings.SplitN(p, "=", 2)
						if len(kv) == 2 {
							inputValues[kv[0]] = kv[1]
						}
					}
				}
				resp, statusCode, err := client.InvokeEndpoint(cfg.BaseURL, ep, inputValues)
				statusColor := "green" // default success
				output.BorderStyle.Fg = ui.ColorGreen

				if statusCode >= 400 {
					statusColor = "red"
					output.BorderStyle.Fg = ui.ColorRed
				}

				if err != nil {
					output.Text = fmt.Sprintf("Error: %s ([Status: %d](fg:%s))", err.Error(), statusCode, statusColor)
					output.BorderStyle.Fg = ui.ColorRed
				} else {
					output.Text = fmt.Sprintf("[Status: %d](fg:%s)\n\n%s", statusCode, statusColor, resp)
				}
				userInput = ""
				input.Text = ""
			case "i":
				inputMode = true
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
			ui.Render(list, input, output)
		}
	}
}
