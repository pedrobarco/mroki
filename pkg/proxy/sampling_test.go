package proxy_test

import (
	"testing"

	"github.com/pedrobarco/mroki/pkg/proxy"
	"github.com/stretchr/testify/assert"
)

func TestNewSamplingRate(t *testing.T) {
	assert := assert.New(t)

	_, err := proxy.NewSamplingRate(0.5)
	assert.Nil(err)

	_, err = proxy.NewSamplingRate(2)
	assert.NotNil(err)
}

func TestSamplingRateSample(t *testing.T) {
	assert := assert.New(t)

	// Test with a sampling rate of 0.9
	sr, err := proxy.NewSamplingRate(0.9)
	assert.Nil(err)

	count := 100
	sampled := 0
	for range count {
		if sr.Sample() {
			sampled++
		}
	}

	// Check that the number of sampled requests is within the expected range
	assert.GreaterOrEqual(float64(sampled), 0.8*float64(count))
	assert.LessOrEqual(float64(sampled), 1.0*float64(count))
}

func TestSamplingRateString(t *testing.T) {
	assert := assert.New(t)

	sr, err := proxy.NewSamplingRate(0.567)
	assert.Nil(err)
	assert.Equal("0.57", sr.String())
}
