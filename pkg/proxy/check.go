package proxy

import "net/http"

// CheckFunc returns true if the request should be proxied to shadow
type CheckFunc func(r *http.Request) bool

// SamplingRateCheck skips shadow based on sampling probability.
// rate == nil means always sample (always returns true).
// rate=1.0 and rate=0.0 use fast paths without RNG computation.
func SamplingRateCheck(rate *SamplingRate) CheckFunc {
	return func(r *http.Request) bool {
		if rate == nil {
			return true
		}
		return rate.Sample()
	}
}
