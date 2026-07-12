package testdata

import "context"

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
	// Operation is the math operation to perform.
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

//tool:service WeatherService
type WeatherService interface {
	//tool:tool
	//tool:desc Get the current weather for a location
	GetWeather(ctx context.Context, req GetWeatherRequest) (GetWeatherResponse, error)

	//tool:tool
	//tool:desc Perform a calculation
	Calculate(ctx context.Context, req CalculateRequest) (CalculateResponse, error)
}
