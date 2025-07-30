package proxy

import (
	"fmt"
	"math/rand/v2"
)

type SamplingRate float64

func NewSamplingRate(rate float64) (*SamplingRate, error) {
	if rate < 0 || rate > 1 {
		return nil, fmt.Errorf("invalid sampling rate: %f, must be between 0 and 1", rate)
	}
	sr := SamplingRate(rate)
	return &sr, nil
}

// Sample decides whether to sample based on the sampling rate.
// It returns true if the request should be sampled, false otherwise.
func (s SamplingRate) Sample() bool {
	return rand.Float64() < float64(s)
}

// IsZero checks if the sampling rate is zero.
func (s SamplingRate) IsZero() bool {
	return s == 0
}

// String returns the sampling rate as a formatted string.
func (s SamplingRate) String() string {
	return fmt.Sprintf("%.2f", float64(s))
}
