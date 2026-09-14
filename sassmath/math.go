package sassmath

import "math"

var CustomPow func(x, y float64) float64
var CustomAtan2 func(y, x float64) float64
var CustomSin func(x float64) float64
var CustomCos func(x float64) float64
var CustomTan func(x float64) float64
var CustomAtan func(x float64) float64
var CustomAsin func(x float64) float64
var CustomAcos func(x float64) float64
var CustomLog func(x float64) float64
var CustomSqrt func(x float64) float64
var CustomAbs func(x float64) float64

func Pow(x, y float64) float64 {
	if CustomPow != nil {
		return CustomPow(x, y)
	}
	return math.Pow(x, y)
}

func Atan2(y, x float64) float64 {
	if CustomAtan2 != nil {
		return CustomAtan2(y, x)
	}
	return math.Atan2(y, x)
}

func Sin(x float64) float64 {
	if CustomSin != nil {
		return CustomSin(x)
	}
	return math.Sin(x)
}

func Cos(x float64) float64 {
	if CustomCos != nil {
		return CustomCos(x)
	}
	return math.Cos(x)
}

func Tan(x float64) float64 {
	if CustomTan != nil {
		return CustomTan(x)
	}
	return math.Tan(x)
}

func Atan(x float64) float64 {
	if CustomAtan != nil {
		return CustomAtan(x)
	}
	return math.Atan(x)
}

func Asin(x float64) float64 {
	if CustomAsin != nil {
		return CustomAsin(x)
	}
	return math.Asin(x)
}

func Acos(x float64) float64 {
	if CustomAcos != nil {
		return CustomAcos(x)
	}
	return math.Acos(x)
}

func Log(x float64) float64 {
	if CustomLog != nil {
		return CustomLog(x)
	}
	return math.Log(x)
}

func Sqrt(x float64) float64 {
	if CustomSqrt != nil {
		return CustomSqrt(x)
	}
	return math.Sqrt(x)
}

func Abs(x float64) float64 {
	if CustomAbs != nil {
		return CustomAbs(x)
	}
	return math.Abs(x)
}
