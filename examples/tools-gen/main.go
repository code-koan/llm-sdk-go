// Code generation workflow example: demo.
//
// This file demonstrates using the generated WeatherServiceHandler.
// Run "go generate" in this directory first, then "go run .".
package main

import (
	"context"
	"encoding/json"
	"fmt"
)

// ---- Mock implementation ----

// weatherService is a mock implementation of WeatherService.
type weatherService struct{}

// GetWeather returns mock weather data.
func (s *weatherService) GetWeather(_ context.Context, req GetWeatherRequest) (GetWeatherResponse, error) {
	conditions := map[string]string{
		"Tokyo":    "sunny",
		"Paris":    "rainy",
		"New York": "cloudy",
		"London":   "foggy",
	}
	cond, ok := conditions[req.Location]
	if !ok {
		cond = "clear"
	}
	return GetWeatherResponse{Temperature: 22.5, Condition: cond}, nil
}

// Calculate performs the requested arithmetic operation.
func (s *weatherService) Calculate(_ context.Context, req CalculateRequest) (CalculateResponse, error) {
	switch req.Operation {
	case "add":
		return CalculateResponse{Result: req.A + req.B}, nil
	case "subtract":
		return CalculateResponse{Result: req.A - req.B}, nil
	case "multiply":
		return CalculateResponse{Result: req.A * req.B}, nil
	case "divide":
		if req.B == 0 {
			return CalculateResponse{}, fmt.Errorf("division by zero")
		}
		return CalculateResponse{Result: req.A / req.B}, nil
	default:
		return CalculateResponse{}, fmt.Errorf("unknown operation: %s", req.Operation)
	}
}

// ---- Demonstration ----

func main() {
	impl := &weatherService{}
	handler := &WeatherServiceHandler{Impl: impl}
	ctx := context.Background()

	// 1. List available MCP tools.
	fmt.Println("=== MCPToolsList ===")
	tools := handler.MCPToolsList()
	for _, t := range tools {
		name, _ := t["name"].(string)
		desc, _ := t["description"].(string)
		fmt.Printf("  name: %q\n  description: %s\n\n", name, desc)
	}

	// 2. Execute a valid tool call: get_weather.
	fmt.Println("=== Execute: get_weather ===")
	args, _ := json.Marshal(GetWeatherRequest{Location: "Tokyo", Unit: "celsius"})
	result, err := handler.Execute(ctx, "get_weather", args)
	if err != nil {
		fmt.Printf("  Error: %v\n\n", err)
	} else {
		var w GetWeatherResponse
		_ = json.Unmarshal([]byte(result), &w)
		fmt.Printf("  Weather in Tokyo: %.1f C, %s\n\n", w.Temperature, w.Condition)
	}

	// 3. Execute another valid tool call: calculate.
	fmt.Println("=== Execute: calculate (5 + 3) ===")
	args, _ = json.Marshal(CalculateRequest{Operation: "add", A: 5, B: 3})
	result, err = handler.Execute(ctx, "calculate", args)
	if err != nil {
		fmt.Printf("  Error: %v\n\n", err)
	} else {
		var c CalculateResponse
		_ = json.Unmarshal([]byte(result), &c)
		fmt.Printf("  5 + 3 = %.0f\n\n", c.Result)
	}

	// 4. Execute an unknown tool (error case).
	fmt.Println("=== Execute: unknown_tool (error case) ===")
	result, err = handler.Execute(ctx, "unknown_tool", nil)
	if err != nil {
		fmt.Printf("  Expected error: %v\n\n", err)
	} else {
		fmt.Printf("  Result: %s\n\n", result)
	}
}
