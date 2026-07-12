package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun_DryRun(t *testing.T) {
	fixture := filepath.Join("..", "..", "internal", "codegen", "testdata", "weather.go")

	var buf bytes.Buffer
	saved := stdout
	stdout = &buf
	defer func() { stdout = saved }()

	err := run([]string{"-source", fixture, "-dry-run"})
	require.NoError(t, err)

	output := buf.String()
	require.Contains(t, output, "func WeatherServiceTools()")
	require.Contains(t, output, "type WeatherServiceHandler struct")
	require.Contains(t, output, "func (h *WeatherServiceHandler) Execute")
}

func TestRun_FileOutput(t *testing.T) {
	fixture := filepath.Join("..", "..", "internal", "codegen", "testdata", "weather.go")
	outputDir := t.TempDir()

	err := run([]string{"-source", fixture, "-output", outputDir})
	require.NoError(t, err)

	genFile := filepath.Join(outputDir, "weather_service.gen.go")
	info, err := os.Stat(genFile)
	require.NoError(t, err, "generated file should exist")
	require.Greater(t, info.Size(), int64(0), "generated file should not be empty")

	data, err := os.ReadFile(genFile)
	require.NoError(t, err)
	require.Contains(t, string(data), "func WeatherServiceTools()")
	require.Contains(t, string(data), "type WeatherServiceHandler struct")
}

func TestRun_MissingSource(t *testing.T) {
	err := run([]string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "-source is required")
}
