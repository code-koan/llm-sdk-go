package codegen

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGenerateGolden regenerates the golden file. Run manually:
//
//	UPDATE_GOLDEN=1 go test -run Golden -v -count=1
func TestGenerateGolden(t *testing.T) {
	if os.Getenv("UPDATE_GOLDEN") != "1" {
		t.Skip("set UPDATE_GOLDEN=1 to regenerate golden")
	}
	svc, err := ParseService("testdata/weather.go")
	require.NoError(t, err)

	f, err := os.Create("testdata/weather.gen.golden")
	require.NoError(t, err)
	defer f.Close()

	err = Generate(f, "testdata", svc)
	require.NoError(t, err)
}

func TestEmit_GoldenFile(t *testing.T) {
	svc, err := ParseService("testdata/weather.go")
	require.NoError(t, err)

	var buf bytes.Buffer
	err = Generate(&buf, "testdata", svc)
	require.NoError(t, err)

	golden, err := os.ReadFile("testdata/weather.gen.golden")
	require.NoError(t, err)

	require.Equal(t, string(golden), buf.String(), "generated code differs from golden file")
}

func TestEmit_Compiles(t *testing.T) {
	svc, err := ParseService("testdata/weather.go")
	require.NoError(t, err)

	dir := t.TempDir()

	// Copy the source file (with types) to temp dir
	src, err := os.ReadFile("testdata/weather.go")
	require.NoError(t, err)

	// Modify package declaration from "testdata" to "main" for standalone compilation
	src = bytes.Replace(src, []byte("package testdata"), []byte("package main"), 1)
	err = os.WriteFile(filepath.Join(dir, "weather.go"), src, 0o644)
	require.NoError(t, err)

	// Generate .gen.go into temp dir with package "main"
	err = GenerateFile(filepath.Join(dir, "weather.gen.go"), "main", svc)
	require.NoError(t, err)

	// Initialize go module in temp dir with replace for llm-sdk-go
	initCmd := exec.Command("go", "mod", "init", "testmodule")
	initCmd.Dir = dir
	out, err := initCmd.CombinedOutput()
	require.NoError(t, err, "go mod init: %s", out)

	rootCmd := exec.Command("go", "env", "GOMOD")
	rootCmd.Dir = "."
	rootOut, _ := rootCmd.Output()
	modFile := strings.TrimSpace(string(rootOut))
	modDir := filepath.Dir(modFile)

	editCmd := exec.Command("go", "mod", "edit", "-replace", "github.com/code-koan/llm-sdk-go="+modDir)
	editCmd.Dir = dir
	out, err = editCmd.CombinedOutput()
	require.NoError(t, err, "go mod edit: %s", out)

	// Tidy to pull in required dependencies via the replace directive
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = dir
	out, err = tidyCmd.CombinedOutput()
	require.NoError(t, err, "go mod tidy: %s", out)

	// Write a stub main so go build works with package main
	err = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)
	require.NoError(t, err)

	// go build from within the temp dir (uses local go.mod)
	buildCmd := exec.Command("go", "build", "-o", "out", ".")
	buildCmd.Dir = dir
	out, err = buildCmd.CombinedOutput()
	require.NoError(t, err, "generated code failed to compile:\n%s", out)
}

func TestEmit_Behavior(t *testing.T) {
	svc, err := ParseService("testdata/weather.go")
	require.NoError(t, err)

	dir := t.TempDir()

	// Copy and rename source to "main" package
	src, err := os.ReadFile("testdata/weather.go")
	require.NoError(t, err)
	src = bytes.Replace(src, []byte("package testdata"), []byte("package main"), 1)
	err = os.WriteFile(filepath.Join(dir, "weather.go"), src, 0o644)
	require.NoError(t, err)

	// Generate .gen.go
	err = GenerateFile(filepath.Join(dir, "weather.gen.go"), "main", svc)
	require.NoError(t, err)

	// Write a test file that exercises the generated handler
	testSrc := `
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "testing"
)

// mockService implements WeatherService
type mockService struct {
    getWeatherFn func(ctx context.Context, req GetWeatherRequest) (GetWeatherResponse, error)
    calculateFn  func(ctx context.Context, req CalculateRequest) (CalculateResponse, error)
}

func (m *mockService) GetWeather(ctx context.Context, req GetWeatherRequest) (GetWeatherResponse, error) {
    return m.getWeatherFn(ctx, req)
}

func (m *mockService) Calculate(ctx context.Context, req CalculateRequest) (CalculateResponse, error) {
    return m.calculateFn(ctx, req)
}

func TestHandler_Execute(t *testing.T) {
    mock := &mockService{
        getWeatherFn: func(ctx context.Context, req GetWeatherRequest) (GetWeatherResponse, error) {
            if req.Location != "Paris" {
                t.Errorf("expected Location=Paris, got %s", req.Location)
            }
            if req.Unit != "celsius" {
                t.Errorf("expected Unit=celsius, got %s", req.Unit)
            }
            return GetWeatherResponse{Temperature: 22.5, Condition: "sunny"}, nil
        },
        calculateFn: func(ctx context.Context, req CalculateRequest) (CalculateResponse, error) {
            if req.Operation != "add" {
                return CalculateResponse{}, fmt.Errorf("expected operation=add, got %s", req.Operation)
            }
            if req.A != 3 || req.B != 4 {
                return CalculateResponse{}, fmt.Errorf("expected a=3, b=4, got a=%f, b=%f", req.A, req.B)
            }
            return CalculateResponse{Result: req.A + req.B}, nil
        },
    }
    handler := &WeatherServiceHandler{Impl: mock}

    // Test valid tool call
    result, err := handler.Execute(context.Background(), "get_weather", json.RawMessage(` + "`" + `{"location":"Paris","unit":"celsius"}` + "`" + `))
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    var resp GetWeatherResponse
    if err := json.Unmarshal([]byte(result), &resp); err != nil {
        t.Fatalf("failed to unmarshal result: %v", err)
    }
    if resp.Temperature != 22.5 {
        t.Errorf("expected 22.5, got %f", resp.Temperature)
    }
    if resp.Condition != "sunny" {
        t.Errorf("expected sunny, got %s", resp.Condition)
    }

    // Test unknown tool
    _, err = handler.Execute(context.Background(), "unknown_tool", json.RawMessage(` + "`" + `{}` + "`" + `))
    if err == nil {
        t.Fatal("expected error for unknown tool")
    }

    // Test MCPToolsList
    tools := handler.MCPToolsList()
    if len(tools) != 2 {
        t.Fatalf("expected 2 tools, got %d", len(tools))
    }
    if tools[0]["name"] != "get_weather" {
        t.Errorf("expected get_weather, got %v", tools[0]["name"])
    }

    // Test MCPToolsCall
    mcpResult, err := handler.MCPToolsCall(context.Background(), "calculate", json.RawMessage(` + "`" + `{"operation":"add","a":3,"b":4}` + "`" + `))
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(mcpResult.Content) == 0 || mcpResult.Content[0].Type != "text" {
        t.Fatal("expected text content in MCP result")
    }
}
`
	err = os.WriteFile(filepath.Join(dir, "mock_test.go"), []byte(testSrc), 0o644)
	require.NoError(t, err)

	// Initialize go module in temp dir
	initCmd := exec.Command("go", "mod", "init", "testmodule")
	initCmd.Dir = dir
	_ = initCmd.Run() // ignore error

	// Add replace directive for llm-sdk-go
	// Find module root
	rootCmd := exec.Command("go", "env", "GOMOD")
	rootCmd.Dir = "."
	rootOut, _ := rootCmd.Output()
	modFile := strings.TrimSpace(string(rootOut))
	modDir := filepath.Dir(modFile)

	editCmd := exec.Command("go", "mod", "edit", "-replace", "github.com/code-koan/llm-sdk-go="+modDir)
	editCmd.Dir = dir
	out, err := editCmd.CombinedOutput()
	require.NoError(t, err, "go mod edit: %s", out)

	// Tidy to pull in required dependencies via the replace directive
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = dir
	out, err = tidyCmd.CombinedOutput()
	require.NoError(t, err, "go mod tidy: %s", out)

	// Run go test from within temp dir
	testCmd := exec.Command("go", "test", "-v", "-count=1", ".")
	testCmd.Dir = dir
	out, err = testCmd.CombinedOutput()
	require.NoError(t, err, "behavioral test failed:\n%s", out)
	t.Logf("Behavioral test output:\n%s", out)
}
