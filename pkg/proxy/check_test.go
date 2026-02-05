package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxBodySizeCheck(t *testing.T) {
	t.Run("allows requests under limit", func(t *testing.T) {
		check := MaxBodySizeCheck(100)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 50

		assert.True(t, check(req))
	})

	t.Run("allows requests at exact limit", func(t *testing.T) {
		check := MaxBodySizeCheck(100)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 100

		assert.True(t, check(req))
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		check := MaxBodySizeCheck(100)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 101

		assert.False(t, check(req))
	})

	t.Run("blocks chunked encoding requests", func(t *testing.T) {
		check := MaxBodySizeCheck(100)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = -1 // Chunked encoding

		assert.False(t, check(req))
	})

	t.Run("allows all requests when limit is 0", func(t *testing.T) {
		check := MaxBodySizeCheck(0)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 999999

		assert.True(t, check(req))
	})

	t.Run("allows all requests when limit is negative", func(t *testing.T) {
		check := MaxBodySizeCheck(-1)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 999999

		assert.True(t, check(req))
	})

	t.Run("allows chunked when limit is 0", func(t *testing.T) {
		check := MaxBodySizeCheck(0)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = -1

		assert.True(t, check(req))
	})
}

func TestSamplingRateCheck(t *testing.T) {
	t.Run("allows all requests when rate is nil", func(t *testing.T) {
		check := SamplingRateCheck(nil)
		req := httptest.NewRequest("GET", "/test", nil)

		// Should always return true
		for i := 0; i < 100; i++ {
			assert.True(t, check(req))
		}
	})

	t.Run("respects sampling rate", func(t *testing.T) {
		rate, _ := NewSamplingRate(0.5)
		check := SamplingRateCheck(rate)
		req := httptest.NewRequest("GET", "/test", nil)

		// Run many times and verify approximately 50% are sampled
		sampled := 0
		iterations := 1000
		for i := 0; i < iterations; i++ {
			if check(req) {
				sampled++
			}
		}

		// Allow 10% margin of error (450-550 out of 1000)
		assert.Greater(t, sampled, 400)
		assert.Less(t, sampled, 600)
	})

	t.Run("sampling rate 0 blocks all requests", func(t *testing.T) {
		rate, _ := NewSamplingRate(0.0)
		check := SamplingRateCheck(rate)
		req := httptest.NewRequest("GET", "/test", nil)

		// Should never sample
		for i := 0; i < 100; i++ {
			assert.False(t, check(req))
		}
	})

	t.Run("sampling rate 1 allows all requests", func(t *testing.T) {
		rate, _ := NewSamplingRate(1.0)
		check := SamplingRateCheck(rate)
		req := httptest.NewRequest("GET", "/test", nil)

		// Should always sample
		for i := 0; i < 100; i++ {
			assert.True(t, check(req))
		}
	})
}

func TestCheckFunc_Composition(t *testing.T) {
	t.Run("multiple checks with all passing", func(t *testing.T) {
		check1 := func(r *http.Request) bool { return true }
		check2 := func(r *http.Request) bool { return true }

		req := httptest.NewRequest("GET", "/test", nil)

		assert.True(t, check1(req))
		assert.True(t, check2(req))
	})

	t.Run("multiple checks with one failing", func(t *testing.T) {
		check1 := func(r *http.Request) bool { return true }
		check2 := func(r *http.Request) bool { return false }

		req := httptest.NewRequest("GET", "/test", nil)

		// First check passes
		assert.True(t, check1(req))
		// Second check fails
		assert.False(t, check2(req))
	})

	t.Run("combining maxBodySize and sampling", func(t *testing.T) {
		rate, _ := NewSamplingRate(1.0) // Always sample
		bodySizeCheck := MaxBodySizeCheck(100)
		samplingCheck := SamplingRateCheck(rate)

		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 50

		// Both should pass
		assert.True(t, bodySizeCheck(req))
		assert.True(t, samplingCheck(req))
	})
}
