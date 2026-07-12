// Code generation workflow example: service definition.
//
// This file is the input source for llm-tools code generation.
// Run "go generate" to produce weather_service.gen.go, then
// run "go run ." to see the generated dispatch in action.
//
//go:generate go run github.com/code-koan/llm-sdk-go/cmd/llm-tools -source service.go
package main

import "context"

// ---- Request / Response types ----

// GetWeatherRequest is the request for getting weather.
type GetWeatherRequest struct {
	// Location is the city or region to get weather for.
	Location string `json:"location"`
	// Unit is the temperature unit.
	//tool:enum celsius,fahrenheit
	Unit string `json:"unit,omitempty"`
}

// GetWeatherResponse is the response containing weather data.
type GetWeatherResponse struct {
	// Temperature is the current temperature.
	Temperature float64 `json:"temperature"`
	// Condition describes the weather condition.
	Condition string `json:"condition"`
}

// CalculateRequest is the request for a calculation.
type CalculateRequest struct {
	// Operation is the arithmetic operation: add, subtract, multiply, divide.
	Operation string `json:"operation"`
	// A is the first operand.
	A float64 `json:"a"`
	// B is the second operand.
	B float64 `json:"b"`
}

// CalculateResponse is the response containing the result.
type CalculateResponse struct {
	// Result is the computed result.
	Result float64 `json:"result"`
}

// ---- Service interface (source of truth for code generation) ----

//tool:service WeatherService
type WeatherService interface {
	//tool:tool
	//tool:desc Get the current weather for a location
	GetWeather(ctx context.Context, req GetWeatherRequest) (GetWeatherResponse, error)

	//tool:tool
	//tool:desc Perform a mathematical calculation
	Calculate(ctx context.Context, req CalculateRequest) (CalculateResponse, error)
}
