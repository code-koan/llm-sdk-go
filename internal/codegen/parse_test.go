package codegen

import (
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseService_Basic(t *testing.T) {
	t.Parallel()
	si, err := ParseService("testdata/weather.go")
	require.NoError(t, err)
	require.Equal(t, "WeatherService", si.Name)
	require.Len(t, si.Methods, 2)

	m0 := si.Methods[0]
	require.Equal(t, "GetWeather", m0.GoName)
	require.Equal(t, "get_weather", m0.ToolName)
	require.Equal(t, "Get the current weather for a location", m0.Desc)

	m1 := si.Methods[1]
	require.Equal(t, "Calculate", m1.GoName)
	require.Equal(t, "calculate", m1.ToolName)
	require.Equal(t, "Perform a calculation", m1.Desc)
}

func TestParseService_Types(t *testing.T) {
	t.Parallel()
	si, err := ParseService("testdata/weather.go")
	require.NoError(t, err)
	require.Len(t, si.Methods, 2)

	// -- GetWeather request/response types --
	gw := si.Methods[0]

	reqNamed, ok := gw.ReqType.(*types.Named)
	require.True(t, ok, "ReqType should be *types.Named")
	require.Equal(t, "GetWeatherRequest", reqNamed.Obj().Name())
	reqStruct, ok := reqNamed.Underlying().(*types.Struct)
	require.True(t, ok, "underlying type should be *types.Struct")
	require.Equal(t, "Location", reqStruct.Field(0).Name())
	require.Equal(t, "Unit", reqStruct.Field(1).Name())

	respNamed, ok := gw.RespType.(*types.Named)
	require.True(t, ok, "RespType should be *types.Named")
	require.Equal(t, "GetWeatherResponse", respNamed.Obj().Name())
	respStruct, ok := respNamed.Underlying().(*types.Struct)
	require.True(t, ok, "underlying type should be *types.Struct")
	require.Equal(t, "Temperature", respStruct.Field(0).Name())
	require.Equal(t, "Condition", respStruct.Field(1).Name())

	// -- Calculate request types --
	calc := si.Methods[1]

	calcReq, ok := calc.ReqType.(*types.Named)
	require.True(t, ok, "ReqType should be *types.Named")
	require.Equal(t, "CalculateRequest", calcReq.Obj().Name())
	calcStruct, ok := calcReq.Underlying().(*types.Struct)
	require.True(t, ok, "underlying type should be *types.Struct")
	require.Equal(t, "Operation", calcStruct.Field(0).Name())
	require.Equal(t, "A", calcStruct.Field(1).Name())
	require.Equal(t, "B", calcStruct.Field(2).Name())
}

func TestParseService_FieldComments(t *testing.T) {
	t.Parallel()
	si, err := ParseService("testdata/weather.go")
	require.NoError(t, err)
	require.Len(t, si.Methods, 2)

	fc := si.Methods[0].FieldComments

	unitComment, ok := fc["GetWeatherRequest.Unit"]
	require.True(t, ok, "expected key GetWeatherRequest.Unit")
	require.Contains(t, unitComment, "//tool:enum celsius,fahrenheit")

	locComment, ok := fc["GetWeatherRequest.Location"]
	require.True(t, ok, "expected key GetWeatherRequest.Location")
	require.Contains(t, locComment, "Location is the city")
}

func TestParseService_NoServiceAnnotation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "noservice.go")
	src := `package test

type Foo struct{}
`
	err := os.WriteFile(tmpFile, []byte(src), 0644)
	require.NoError(t, err)

	_, err = ParseService(tmpFile)
	require.Error(t, err)
}
