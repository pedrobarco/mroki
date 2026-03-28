package caddymodule

import (
	"testing"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMrokiGate_Validate_required_fields(t *testing.T) {
	t.Run("missing live URL", func(t *testing.T) {
		m := MrokiGate{RawShadow: "http://shadow:8080"}
		err := m.Validate()
		assert.ErrorIs(t, err, ErrRequiredLiveURL)
	})

	t.Run("missing shadow URL", func(t *testing.T) {
		m := MrokiGate{RawLive: "http://live:8080"}
		err := m.Validate()
		assert.ErrorIs(t, err, ErrRequiredShadowURL)
	})

	t.Run("valid minimal config", func(t *testing.T) {
		m := MrokiGate{
			RawLive:   "http://live:8080",
			RawShadow: "http://shadow:8080",
		}
		err := m.Validate()
		require.NoError(t, err)
		assert.NotNil(t, m.proxy)
	})
}

func TestMrokiGate_Validate_sampling_rate(t *testing.T) {
	t.Run("valid sampling rate", func(t *testing.T) {
		rate := "0.5"
		m := MrokiGate{
			RawLive:      "http://live:8080",
			RawShadow:    "http://shadow:8080",
			SamplingRate: &rate,
		}
		err := m.Validate()
		require.NoError(t, err)
	})

	t.Run("invalid sampling rate", func(t *testing.T) {
		rate := "not-a-number"
		m := MrokiGate{
			RawLive:      "http://live:8080",
			RawShadow:    "http://shadow:8080",
			SamplingRate: &rate,
		}
		err := m.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid sampling rate")
	})
}

func TestMrokiGate_Validate_max_body_size(t *testing.T) {
	size := "1048576"
	m := MrokiGate{
		RawLive:        "http://live:8080",
		RawShadow:      "http://shadow:8080",
		RawMaxBodySize: &size,
	}
	err := m.Validate()
	require.NoError(t, err)
}

func TestMrokiGate_Validate_diff_options(t *testing.T) {
	ignored := "timestamp,created_at"
	included := "user,order"
	tolerance := "0.001"
	m := MrokiGate{
		RawLive:            "http://live:8080",
		RawShadow:          "http://shadow:8080",
		DiffIgnoredFields:  &ignored,
		DiffIncludedFields: &included,
		DiffFloatTolerance: &tolerance,
	}
	err := m.Validate()
	require.NoError(t, err)
}

func TestUnmarshalCaddyfile_all_directives(t *testing.T) {
	input := `mroki_gate {
		live http://live:8080
		shadow http://shadow:8080
		sampling_rate 0.5
		live_timeout 3s
		shadow_timeout 10s
		max_body_size 1048576
		diff_ignored_fields timestamp,created_at
		diff_included_fields user,order
		diff_float_tolerance 0.001
	}`

	d := caddyfile.NewTestDispenser(input)
	var m MrokiGate
	err := m.UnmarshalCaddyfile(d)

	require.NoError(t, err)
	assert.Equal(t, "http://live:8080", m.RawLive)
	assert.Equal(t, "http://shadow:8080", m.RawShadow)
	require.NotNil(t, m.SamplingRate)
	assert.Equal(t, "0.5", *m.SamplingRate)
	require.NotNil(t, m.RawLiveTimeout)
	assert.Equal(t, "3s", *m.RawLiveTimeout)
	require.NotNil(t, m.RawShadowTimeout)
	assert.Equal(t, "10s", *m.RawShadowTimeout)
	require.NotNil(t, m.RawMaxBodySize)
	assert.Equal(t, "1048576", *m.RawMaxBodySize)
	require.NotNil(t, m.DiffIgnoredFields)
	assert.Equal(t, "timestamp,created_at", *m.DiffIgnoredFields)
	require.NotNil(t, m.DiffIncludedFields)
	assert.Equal(t, "user,order", *m.DiffIncludedFields)
	require.NotNil(t, m.DiffFloatTolerance)
	assert.Equal(t, "0.001", *m.DiffFloatTolerance)
}

func TestUnmarshalCaddyfile_unknown_property(t *testing.T) {
	input := `mroki_gate {
		live http://live:8080
		shadow http://shadow:8080
		api_url http://api:8081
	}`

	d := caddyfile.NewTestDispenser(input)
	var m MrokiGate
	err := m.UnmarshalCaddyfile(d)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown property 'api_url'")
}


func TestMrokiGate_Validate_invalid_float_tolerance(t *testing.T) {
	tolerance := "not-a-number"
	m := MrokiGate{
		RawLive:            "http://live:8080",
		RawShadow:          "http://shadow:8080",
		DiffFloatTolerance: &tolerance,
	}
	err := m.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid diff_float_tolerance")
}

func TestMrokiGate_Validate_invalid_live_timeout(t *testing.T) {
	timeout := "not-a-duration"
	m := MrokiGate{
		RawLive:        "http://live:8080",
		RawShadow:      "http://shadow:8080",
		RawLiveTimeout: &timeout,
	}
	err := m.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid live timeout")
}

func TestMrokiGate_Validate_invalid_shadow_timeout(t *testing.T) {
	timeout := "bad"
	m := MrokiGate{
		RawLive:          "http://live:8080",
		RawShadow:        "http://shadow:8080",
		RawShadowTimeout: &timeout,
	}
	err := m.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid shadow timeout")
}

func TestMrokiGate_Validate_invalid_max_body_size(t *testing.T) {
	size := "not-a-number"
	m := MrokiGate{
		RawLive:        "http://live:8080",
		RawShadow:      "http://shadow:8080",
		RawMaxBodySize: &size,
	}
	err := m.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid max_body_size")
}

func TestMrokiGate_Validate_sampling_rate_out_of_range(t *testing.T) {
	rate := "1.5"
	m := MrokiGate{
		RawLive:      "http://live:8080",
		RawShadow:    "http://shadow:8080",
		SamplingRate: &rate,
	}
	err := m.Validate()
	assert.Error(t, err)
}

func TestUnmarshalCaddyfile_minimal(t *testing.T) {
	input := `mroki_gate {
		live http://live:8080
		shadow http://shadow:8080
	}`

	d := caddyfile.NewTestDispenser(input)
	var m MrokiGate
	err := m.UnmarshalCaddyfile(d)

	require.NoError(t, err)
	assert.Equal(t, "http://live:8080", m.RawLive)
	assert.Equal(t, "http://shadow:8080", m.RawShadow)
	assert.Nil(t, m.SamplingRate)
	assert.Nil(t, m.RawLiveTimeout)
	assert.Nil(t, m.RawShadowTimeout)
	assert.Nil(t, m.RawMaxBodySize)
	assert.Nil(t, m.DiffIgnoredFields)
	assert.Nil(t, m.DiffIncludedFields)
	assert.Nil(t, m.DiffFloatTolerance)
}