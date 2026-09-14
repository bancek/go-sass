package sassmathcgo

// #cgo LDFLAGS: -lm
// #include <math.h>
import "C"

import "github.com/bancek/go-sass/sassmath"

func SassMathCgo() {
	sassmath.CustomPow = func(x, y float64) float64 {
		return float64(C.pow(C.double(x), C.double(y)))
	}
	sassmath.CustomAtan2 = func(y, x float64) float64 {
		return float64(C.atan2(C.double(y), C.double(x)))
	}
	sassmath.CustomSin = func(x float64) float64 {
		return float64(C.sin(C.double(x)))
	}
	sassmath.CustomCos = func(x float64) float64 {
		return float64(C.cos(C.double(x)))
	}
	sassmath.CustomTan = func(x float64) float64 {
		return float64(C.tan(C.double(x)))
	}
	sassmath.CustomAtan = func(x float64) float64 {
		return float64(C.atan(C.double(x)))
	}
	sassmath.CustomAsin = func(x float64) float64 {
		return float64(C.asin(C.double(x)))
	}
	sassmath.CustomAcos = func(x float64) float64 {
		return float64(C.acos(C.double(x)))
	}
	sassmath.CustomLog = func(x float64) float64 {
		return float64(C.log(C.double(x)))
	}
	sassmath.CustomSqrt = func(x float64) float64 {
		return float64(C.sqrt(C.double(x)))
	}
	sassmath.CustomAbs = func(x float64) float64 {
		return float64(C.fabs(C.double(x)))
	}
}
