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

## How To Use The TUI

**Navigation**
- `j` / `<Down>`: move down
- `k` / `<Up>`: move up
- `<Enter>`: invoke the selected endpoint
- `q` / `<C-c>`: quit

**Editing inputs**
- `b`: edit Base URL
- `H`: edit Headers
- `i`: edit Query Parameters

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
