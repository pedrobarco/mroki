package proxy

import "net/http"

// CheckFunc returns true if the request should be proxied to shadow
type CheckFunc func(r *http.Request) bool

// MaxBodySizeCheck skips shadow for large or chunked bodies
// maxBytes <= 0 means unlimited (always returns true)
func MaxBodySizeCheck(maxBytes int64) CheckFunc {
	return func(r *http.Request) bool {
		if maxBytes <= 0 {
			return true
		}

		// Skip chunked encoding (unknown size)
		if r.ContentLength < 0 {
			return false
		}

		return r.ContentLength <= maxBytes
	}
}

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
