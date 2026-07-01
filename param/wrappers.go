package param

// Int returns an Opt[int] with an included value.
// Usage: param.Int(1024) -> Opt[int]{Value: 1024, status: included}
func Int(i int) Opt[int] { return NewOpt(i) }

// Float returns an Opt[float64] with an included value.
func Float(f float64) Opt[float64] { return NewOpt(f) }

// Bool returns an Opt[bool] with an included value.
func Bool(b bool) Opt[bool] { return NewOpt(b) }

// String returns an Opt[string] with an included value.
func String(s string) Opt[string] { return NewOpt(s) }
