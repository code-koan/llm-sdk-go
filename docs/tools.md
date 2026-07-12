# Tool Code Generation

Use `go generate` to automatically produce Tool schemas, dispatch handlers, and MCP servers from annotated Go interfaces.

## Quick Start

1. Annotate your service interface:

```go
//tool:service WeatherService
type WeatherService interface {
    //tool:tool
    //tool:desc Get the current weather for a location
    GetWeather(ctx context.Context, req GetWeatherRequest) (GetWeatherResponse, error)
}
```

2. Add the generate directive:

```go
//go:generate go run github.com/code-koan/llm-sdk-go/cmd/llm-tools -source service.go
```

3. Run `go generate ./...`

4. Use the generated code:

```go
tools := WeatherServiceTools()                      // []Tool for ChatBuilder
handler := &WeatherServiceHandler{Impl: myService}  // dispatch handler

// LLM tool dispatch
result, _ := handler.Execute(ctx, toolName, arguments)

// MCP server
http.ListenAndServe(":8080", handler.MCPHandler())
```

## Annotation Reference

### Interface-level

| Annotation | Required | Description |
|-----------|----------|-------------|
| `//tool:service <Name>` | Yes | Marks the interface as a tool service. Name used for generated identifiers. |

### Method-level

| Annotation | Required | Description |
|-----------|----------|-------------|
| `//tool:tool` | Yes | Marks the method as a tool |
| `//tool:desc <text>` | No | Tool description (default: empty) |
| `//tool:name <name>` | No | Tool name override (default: snake_case of method name) |

### Field-level (on struct fields)

| Annotation | Required | Description |
|-----------|----------|-------------|
| Doc comment | No | Becomes the field's `description` in JSON Schema |
| `//tool:enum a,b,c` | No | Comma-separated enum values |

## Method Signature

```go
MethodName(ctx context.Context, req T) (R, error)
```

- First param: `context.Context`
- Second param: request struct (must be JSON-serializable)
- Returns: response struct + error

## Generated API

The generated file (`<service>.gen.go`) provides:

| Export | Type | Purpose |
|--------|------|---------|
| `<Name>Tools()` | `[]providers.Tool` | Tool schemas for ChatBuilder |
| `<Name>Handler` | struct | Dispatch handler |
| `Handler.Execute()` | `(ctx, name, args) (string, error)` | Execute a tool call |
| `Handler.MCPToolsList()` | `[]map[string]any` | MCP tools/list response |
| `Handler.MCPToolsCall()` | `(ctx, name, args) (*MCPCallResult, error)` | MCP tools/call execution |
| `Handler.MCPHandler()` | `http.Handler` | Full MCP JSON-RPC 2.0 server |

## CLI Reference

```bash
llm-tools -source <file.go> [-output <dir>] [-dry-run]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-source` | (required) | Input Go source file |
| `-output` | source directory | Output directory |
| `-dry-run` | false | Print to stdout instead of writing |

## Type Mapping

| Go Type | JSON Schema |
|---------|-------------|
| `string` | `"string"` |
| `int`, `int64`, `uint`, etc. | `"integer"` |
| `float32`, `float64` | `"number"` |
| `bool` | `"boolean"` |
| `[]T` | `{"type":"array","items":...}` |
| `struct` | `{"type":"object","properties":...}` |
| `*T` | Recursive, not in `required` |

## See Also

- [examples/tools-gen/](../examples/tools-gen/) — Full working example
