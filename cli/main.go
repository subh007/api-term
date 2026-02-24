package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"google.golang.org/genai"
	"org.subh/api-term/pkgs/ai"
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
	List              *widgets.List
	Output            *widgets.List
	BaseURLWidget     *widgets.Paragraph
	HeadersWidget     *widgets.Paragraph
	Input             *widgets.Paragraph
	BodyWidget        *widgets.Paragraph
	ContentTypeWidget *widgets.Paragraph
	Help              *widgets.Paragraph

	// State
	FocusMode        string
	ShowHelp         bool
	InputMode        bool
	EditTarget       string
	EditBuffer       string
	QueryInput       string
	HeaderInput      string
	BodyInput        string
	ContentTypeInput string
	BaseURL          string
	InputValues      map[string]string
	HeaderValues     map[string]string
	TermWidth        int
	TermHeight       int

	// Gemini State
	ShowGemini   bool
	GeminiZoomed bool
	GeminiWidget *widgets.List
	GeminiInput  *widgets.Paragraph
	GeminiChat   *genai.Chat
	GeminiQuery  string
	GeminiCtx    context.Context
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
	input.Text = ""
	input.BorderStyle.Fg = ui.ColorCyan

	bodyWidget := widgets.NewParagraph()
	bodyWidget.Title = "Body (Press 'B' to edit)"
	bodyWidget.Text = ""
	bodyWidget.BorderStyle.Fg = ui.ColorCyan

	contentTypeWidget := widgets.NewParagraph()
	contentTypeWidget.Title = "Content-Type (Press 'C' to edit)"
	contentTypeWidget.Text = "application/json"
	contentTypeWidget.BorderStyle.Fg = ui.ColorCyan

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
	  B            Edit Body
	  C            Edit Content-Type
	  g            Toggle Gemini Insights (Tab to focus Gemini/Output)
	  G            Chat with Gemini
	  Z            Zoom/Fullscreen Gemini Insights
	  ? / h        Toggle Help
	  q / <C-c>    Quit
	`
	help.BorderStyle.Fg = ui.ColorYellow

	geminiWidget := widgets.NewList()
	geminiWidget.Title = "Gemini Insights"
	geminiWidget.Rows = []string{}
	geminiWidget.WrapText = true
	geminiWidget.TextStyle = ui.NewStyle(ui.ColorCyan)
	geminiWidget.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorCyan)
	geminiWidget.BorderStyle.Fg = ui.ColorCyan

	geminiInput := widgets.NewParagraph()
	geminiInput.Title = "Gemini Chat (Press 'G' to ask)"
	geminiInput.Text = ""
	geminiInput.BorderStyle.Fg = ui.ColorMagenta

	h := &MainHandler{
		Endpoints:         endpoints,
		GlobalQueryParams: cfg.GlobalQueryParams,
		Config:            cfg,
		List:              list,
		Output:            output,
		BaseURLWidget:     baseURLWidget,
		HeadersWidget:     headersWidget,
		Input:             input,
		BodyWidget:        bodyWidget,
		ContentTypeWidget: contentTypeWidget,
		GeminiWidget:      geminiWidget,
		GeminiInput:       geminiInput,
		Help:              help,
		FocusMode:         "list",
		BaseURL:           cfg.BaseURL,
		ContentTypeInput:  "application/json",
		InputValues:       make(map[string]string),
		HeaderValues:      make(map[string]string),
		GeminiCtx:         context.Background(),
	}

	return h
}

func (h *MainHandler) Init(termWidth, termHeight int) {
	h.Resize(termWidth, termHeight)
}

func (h *MainHandler) Resize(termWidth, termHeight int) {
	h.TermWidth = termWidth
	h.TermHeight = termHeight
	h.updateLayout()
}

func (h *MainHandler) updateLayout() {
	termWidth := h.TermWidth
	termHeight := h.TermHeight

	if h.GeminiZoomed {
		h.GeminiWidget.SetRect(0, 0, termWidth, termHeight-3)
		h.GeminiInput.SetRect(0, termHeight-3, termWidth, termHeight)

		// Hide everything else
		h.List.SetRect(0, 0, 0, 0)
		h.Output.SetRect(0, 0, 0, 0)
		h.BaseURLWidget.SetRect(0, 0, 0, 0)
		h.HeadersWidget.SetRect(0, 0, 0, 0)
		h.Input.SetRect(0, 0, 0, 0)
		h.BodyWidget.SetRect(0, 0, 0, 0)
		h.ContentTypeWidget.SetRect(0, 0, 0, 0)
		h.Help.SetRect(termWidth/4, termHeight/4, 3*termWidth/4, 3*termHeight/4)
		return
	}

	// Determine if we need to show body widget
	showBody := false
	if len(h.Endpoints) > 0 && h.List.SelectedRow < len(h.Endpoints) {
		currEp := h.Endpoints[h.List.SelectedRow]
		if strings.EqualFold(currEp.Method, "POST") || strings.EqualFold(currEp.Method, "PUT") {
			showBody = true
		}
	}

	// Bottom-up allocation
	bottomY := termHeight

	// ContentType (Always allocate space for consistency, though only rendered for POST/PUT)
	h.ContentTypeWidget.SetRect(0, bottomY-3, termWidth, bottomY)
	bottomY -= 3

	// Input
	h.Input.SetRect(0, bottomY-3, termWidth, bottomY)
	bottomY -= 3

	// Check space for Headers
	if bottomY >= 5 { // Need at least 2 lines for List/Output + 3 for Headers
		h.HeadersWidget.SetRect(0, bottomY-3, termWidth, bottomY)
		bottomY -= 3
	} else {
		h.HeadersWidget.SetRect(0, 0, 0, 0) // Hide
	}

	// Check space for BaseURL
	if bottomY >= 5 { // Need at least 2 lines for List/Output + 3 for BaseURL
		h.BaseURLWidget.SetRect(0, bottomY-3, termWidth, bottomY)
		bottomY -= 3
	} else {
		h.BaseURLWidget.SetRect(0, 0, 0, 0) // Hide
	}

	// List and Output share remaining top space
	if bottomY > 0 {
		listHeight := bottomY / 2
		if listHeight < 1 {
			listHeight = 1
		}

		if showBody {
			// Split top area for List and BodyWidget
			h.List.SetRect(0, 0, termWidth/2, listHeight)
			h.BodyWidget.SetRect(termWidth/2, 0, termWidth, listHeight)
		} else {
			// List takes full width
			h.List.SetRect(0, 0, termWidth, listHeight)
			h.BodyWidget.SetRect(0, 0, 0, 0) // Hide
		}

		if h.ShowGemini {
			h.Output.SetRect(0, listHeight, termWidth/2, bottomY)
			h.GeminiWidget.SetRect(termWidth/2, listHeight, termWidth, bottomY-3)
			h.GeminiInput.SetRect(termWidth/2, bottomY-3, termWidth, bottomY)
		} else {
			h.Output.SetRect(0, listHeight, termWidth, bottomY)
			h.GeminiWidget.SetRect(0, 0, 0, 0)
			h.GeminiInput.SetRect(0, 0, 0, 0)
		}
	} else {
		// Extreme fallback
		h.List.SetRect(0, 0, termWidth, 1)
		h.Output.SetRect(0, 0, 0, 0)
	}

	h.Help.SetRect(termWidth/4, termHeight/4, 3*termWidth/4, 3*termHeight/4)
}

func (h *MainHandler) Render() {
	if h.ShowHelp {
		ui.Render(h.Help)
	} else if h.GeminiZoomed {
		h.updateLayout()
		ui.Render(h.GeminiWidget, h.GeminiInput)
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

		h.updateLayout()
		ui.Render(h.List, h.Output, h.BaseURLWidget, h.HeadersWidget, h.Input)

		if strings.EqualFold(currEp.Method, "POST") || strings.EqualFold(currEp.Method, "PUT") {
			ui.Render(h.BodyWidget, h.ContentTypeWidget)
		}

		if h.ShowGemini && !h.GeminiZoomed {
			ui.Render(h.GeminiWidget, h.GeminiInput)
		}
	}
}

func (h *MainHandler) initGemini() {
	if len(h.GeminiWidget.Rows) == 0 || h.GeminiChat == nil {
		h.GeminiWidget.Rows = []string{"Initializing Gemini insights..."}
		h.GeminiWidget.SelectedRow = 0
		ui.Render(h.GeminiWidget)

		go func() {
			clientCtx := h.GeminiCtx
			gClient, err := ai.NewGeminiClient(clientCtx)
			if err != nil {
				h.GeminiWidget.Rows = []string{"Failed to load Gemini API: " + err.Error()}
				ui.Render(h.GeminiWidget)
				return
			}

			chat, err := gClient.CreateChatSession(clientCtx, "gemini-2.5-flash")
			if err != nil {
				h.GeminiWidget.Rows = []string{"Failed to create chat: " + err.Error()}
				ui.Render(h.GeminiWidget)
				return
			}
			h.GeminiChat = chat
			h.GeminiWidget.Rows = []string{"Thinking..."}
			ui.Render(h.GeminiWidget)

			outText := strings.Join(h.Output.Rows, "\n")
			prompt := "Here is the API response:\n" + outText + "\nProvide interesting insights, then ask the user for any recommendations or follow up actions."
			resp, err := chat.SendMessage(clientCtx, genai.Part{Text: prompt})

			if err != nil {
				h.GeminiWidget.Rows = []string{"Error from Gemini: " + err.Error()}
			} else {
				h.GeminiWidget.Rows = append([]string{"Initial Insights:"}, strings.Split(ai.FormatContent(resp), "\n")...)
			}
			h.GeminiWidget.SelectedRow = 0
			ui.Render(h.GeminiWidget)
		}()
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
			case "body":
				h.BodyInput = h.EditBuffer
				h.BodyWidget.Text = h.BodyInput
				h.BodyWidget.BorderStyle.Fg = ui.ColorCyan
			case "content-type":
				h.ContentTypeInput = strings.TrimSpace(h.EditBuffer)
				h.ContentTypeWidget.Text = h.ContentTypeInput
				h.ContentTypeWidget.BorderStyle.Fg = ui.ColorCyan
			case "gemini":
				h.GeminiQuery = strings.TrimSpace(h.EditBuffer)
				h.GeminiInput.Text = h.GeminiQuery
				h.GeminiInput.BorderStyle.Fg = ui.ColorMagenta

				if h.GeminiChat != nil && h.GeminiQuery != "" {
					h.GeminiWidget.Rows = append(h.GeminiWidget.Rows, "You: "+h.GeminiQuery, "Thinking...")
					ui.Render(h.GeminiWidget)

					go func(query string) {
						resp, err := h.GeminiChat.SendMessage(h.GeminiCtx, genai.Part{Text: query})
						h.GeminiWidget.Rows = h.GeminiWidget.Rows[:len(h.GeminiWidget.Rows)-1] // remove Thinking
						if err != nil {
							h.GeminiWidget.Rows = append(h.GeminiWidget.Rows, "Gemini Error: "+err.Error(), "")
						} else {
							h.GeminiWidget.Rows = append(h.GeminiWidget.Rows, "Gemini:", "")
							h.GeminiWidget.Rows = append(h.GeminiWidget.Rows, strings.Split(ai.FormatContent(resp), "\n")...)
							h.GeminiWidget.Rows = append(h.GeminiWidget.Rows, "")
						}
						h.GeminiWidget.SelectedRow = len(h.GeminiWidget.Rows) - 1
						ui.Render(h.GeminiWidget)
					}(h.GeminiQuery)
				}
				h.GeminiInput.Text = ""
				h.GeminiQuery = ""
			}
			h.EditTarget = ""
		case "<Backspace>":
			if len(h.EditBuffer) > 0 {
				h.EditBuffer = h.EditBuffer[:len(h.EditBuffer)-1]
				if h.EditTarget == "baseurl" {
					h.BaseURLWidget.Text = h.EditBuffer
				} else if h.EditTarget == "headers" {
					h.HeadersWidget.Text = h.EditBuffer
				} else if h.EditTarget == "gemini" {
					h.GeminiInput.Text = h.EditBuffer
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
				} else if h.EditTarget == "body" {
					h.BodyWidget.Text = h.EditBuffer
				} else if h.EditTarget == "content-type" {
					h.ContentTypeWidget.Text = h.EditBuffer
				} else if h.EditTarget == "gemini" {
					h.GeminiInput.Text = h.EditBuffer
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
		if h.ShowGemini && h.FocusMode == "output" {
			h.FocusMode = "gemini"
			h.Output.TitleStyle = ui.NewStyle(ui.ColorWhite)
			h.Output.BorderStyle.Fg = ui.ColorWhite
			h.GeminiWidget.TitleStyle = ui.NewStyle(ui.ColorYellow)
			h.GeminiWidget.BorderStyle.Fg = ui.ColorYellow
		} else if h.ShowGemini && h.FocusMode == "gemini" {
			h.FocusMode = "list"
			h.GeminiWidget.TitleStyle = ui.NewStyle(ui.ColorCyan)
			h.GeminiWidget.BorderStyle.Fg = ui.ColorCyan
			h.List.TitleStyle = ui.NewStyle(ui.ColorYellow)
			h.List.BorderStyle.Fg = ui.ColorYellow
		} else if h.FocusMode == "list" {
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
			if h.ShowGemini {
				h.GeminiWidget.TitleStyle = ui.NewStyle(ui.ColorCyan)
				h.GeminiWidget.BorderStyle.Fg = ui.ColorCyan
			}
		}
	case "j", "<Down>":
		if h.FocusMode == "list" {
			if h.List.SelectedRow < len(h.List.Rows)-1 {
				h.List.SelectedRow++
			}
		} else if h.FocusMode == "gemini" {
			if h.GeminiWidget.SelectedRow < len(h.GeminiWidget.Rows)-1 {
				h.GeminiWidget.SelectedRow++
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
		} else if h.FocusMode == "gemini" {
			if h.GeminiWidget.SelectedRow > 0 {
				h.GeminiWidget.SelectedRow--
			}
		} else {
			if h.Output.SelectedRow > 0 {
				h.Output.SelectedRow--
			}
		}
	case "<Enter>":
		if h.FocusMode == "output" || h.FocusMode == "gemini" {
			return false
		}

		// Reset Gemini chat state when a new API call is made
		h.GeminiChat = nil
		h.GeminiWidget.Rows = []string{}
		h.GeminiWidget.SelectedRow = 0
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
		resp, statusCode, err := client.InvokeEndpoint(h.BaseURL, ep, inputValues, headerValues, h.BodyInput, h.ContentTypeInput)
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
	case "B":
		currEp := h.Endpoints[h.List.SelectedRow]
		if strings.EqualFold(currEp.Method, "POST") || strings.EqualFold(currEp.Method, "PUT") {
			h.InputMode = true
			h.EditTarget = "body"
			h.EditBuffer = h.BodyInput
			h.BodyWidget.Text = h.EditBuffer
			h.BodyWidget.BorderStyle.Fg = ui.ColorYellow
			h.Output.BorderStyle.Fg = ui.ColorWhite
		}
	case "C":
		currEp := h.Endpoints[h.List.SelectedRow]
		if strings.EqualFold(currEp.Method, "POST") || strings.EqualFold(currEp.Method, "PUT") {
			h.InputMode = true
			h.EditTarget = "content-type"
			h.EditBuffer = h.ContentTypeInput
			h.ContentTypeWidget.Text = h.EditBuffer
			h.ContentTypeWidget.BorderStyle.Fg = ui.ColorYellow
			h.Output.BorderStyle.Fg = ui.ColorWhite
		}
	case "g":
		h.ShowGemini = !h.ShowGemini
		if h.ShowGemini {
			h.GeminiZoomed = false
			h.FocusMode = "gemini"
			h.List.TitleStyle = ui.NewStyle(ui.ColorWhite)
			h.List.BorderStyle.Fg = ui.ColorWhite
			h.Output.TitleStyle = ui.NewStyle(ui.ColorWhite)
			h.Output.BorderStyle.Fg = ui.ColorWhite
			h.GeminiWidget.TitleStyle = ui.NewStyle(ui.ColorYellow)
			h.GeminiWidget.BorderStyle.Fg = ui.ColorYellow
			h.updateLayout()
			ui.Clear()
			h.Render()

			h.initGemini()
		} else {
			h.FocusMode = "list"
			h.List.TitleStyle = ui.NewStyle(ui.ColorYellow)
			h.List.BorderStyle.Fg = ui.ColorYellow
			h.Output.TitleStyle = ui.NewStyle(ui.ColorWhite)
			h.Output.BorderStyle.Fg = ui.ColorWhite
			h.updateLayout()
			ui.Clear()
		}
	case "G":
		if h.ShowGemini {
			h.InputMode = true
			h.EditTarget = "gemini"
			h.EditBuffer = h.GeminiQuery
			h.GeminiInput.Text = h.EditBuffer
			h.GeminiInput.BorderStyle.Fg = ui.ColorYellow
		}
	case "Z":
		h.GeminiZoomed = !h.GeminiZoomed
		if h.GeminiZoomed {
			if !h.ShowGemini {
				h.ShowGemini = true
			}
			h.FocusMode = "gemini"
			h.GeminiWidget.TitleStyle = ui.NewStyle(ui.ColorYellow)
			h.GeminiWidget.BorderStyle.Fg = ui.ColorYellow
			h.List.TitleStyle = ui.NewStyle(ui.ColorWhite)
			h.List.BorderStyle.Fg = ui.ColorWhite
			h.Output.TitleStyle = ui.NewStyle(ui.ColorWhite)
			h.Output.BorderStyle.Fg = ui.ColorWhite
			h.initGemini()
		}
		h.updateLayout()
		ui.Clear()
		h.Render()
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
