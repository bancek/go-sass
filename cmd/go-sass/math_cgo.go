//go:build cgomath

package main

import "github.com/bancek/go-sass/sassmath/sassmathcgo"

func init() {
	sassmathcgo.SassMathCgo()
}
