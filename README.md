# API-TERM

Terminal UI for exploring OpenAPI-defined HTTP APIs. Load an OpenAPI spec, browse endpoints, set base URL/headers/query params, and invoke requests from the terminal.

**Requirements**
- Go `1.24.5+`
- A terminal that supports interactive TUI rendering

## Quick Start

Run with the bundled sample spec:
```bash
go run ./cli
```

Run with a local OpenAPI file:
```bash
go run ./cli --file /path/to/openapi.yaml
```

Run with one or more OpenAPI URLs:
```bash
go run ./cli --url https://example.com/openapi.yaml --url https://example.com/other.yaml
```

## Docker

Pull the image from GitHub Container Registry:
```bash
docker pull ghcr.io/subh007/api-term:0.2
```

Run with the bundled sample spec:
```bash
docker run -it ghcr.io/subh007/api-term:0.2
```

Run with a local OpenAPI file:
```bash
docker run -it -v /path/to/openapi.yaml:/spec.yaml ghcr.io/subh007/api-term:0.2 --file /spec.yaml
```

Run with an OpenAPI URL:
```bash
docker run -it ghcr.io/subh007/api-term:0.2 --url https://example.com/openapi.yaml
```

## How To Use The TUI

### demo
[![asciicast](https://asciinema.org/a/a9tRxbO1NsCkAymI.svg)](https://asciinema.org/a/a9tRxbO1NsCkAymI)

**Navigation**
- `j` / `<Down>`: move down
- `k` / `<Up>`: move up
- `<Enter>`: invoke the selected endpoint
- `q` / `<C-c>`: quit

**Editing inputs**
- `b`: edit Base URL
- `H`: edit Headers
- `i`: edit Query Parameters
- `B`: edit Body (for POST/PUT)
- `C`: edit Content-Type (for POST/PUT)

**Gemini Insights (AI)**
- `g`: toggle Gemini Insights widget (splits Output view)
- `Tab`: focus the Gemini widget to scroll history
- `G`: chat with Gemini (input query)
- `Z`: zoom/fullscreen the Gemini widget

**Help**
- `?` or `h`: toggle help overlay

### Input Formats

**Base URL**
- Press `b` and enter the base URL, e.g. `http://localhost:8080`

**Headers**
- Press `H` and enter key/value pairs:
  - `Authorization: Bearer TOKEN&X-Env: dev`
  - `Authorization=Bearer TOKEN;X-Env=dev`

**Query parameters**
- Press `i` and enter key/value pairs:
  - `page=1&limit=10`
- If an endpoint has exactly one required param (path or query) and you enter a single value without `=` or `&`, that value is used for the required param.

## OpenAPI Behavior

- Endpoints are populated from the OpenAPI spec.
- Required query parameters are enforced when invoking an endpoint.
- Query parameters are URL-encoded before the request is sent.
- The list view shows query parameter names as `?param1&param2` next to the path.

## Google Gemini API Integrations

You can leverage Google's GenAI directly within the TUI to summarize and analyze API responses!

**Prerequisites:**
You must export your API key to your environment before running `api-term`:
```bash
export GEMINI_API_KEY="your-api-key-here"
```

**How To Use:**
1. Focus an endpoint and trigger a request (`<Enter>`) to get a response.
2. Press `g` to open the Gemini widget. The app will automatically analyze the response body and ask for recommendations or insights.
3. Press `Tab` to navigate to the Gemini widget and use `j`/`k` to scroll through the analysis.
4. Press `G` to type a follow-up query securely to the AI model.
5. Press `Z` to zoom the model securely to the entire terminal window for an immersive chat experience.

## Mock Server (Optional)

This repo includes a mountebank configuration for quick local testing:
```bash
npm install -g mountebank
mb --port 2525
mb --configfile mock-api.json
```

## Notes

- Currently, only `GET` requests are sent.
- Path parameters are filled using values you provide in the input area.
