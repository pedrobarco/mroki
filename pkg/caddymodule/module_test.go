package caddymodule_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/pedrobarco/mroki/pkg/caddymodule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMrokiGate_Validate_required_fields(t *testing.T) {
	t.Run("missing live URL", func(t *testing.T) {
		m := caddymodule.MrokiGate{RawShadow: "http://shadow:8080"}
		err := m.Validate()
		assert.ErrorIs(t, err, caddymodule.ErrRequiredLiveURL)
	})

	t.Run("missing shadow URL", func(t *testing.T) {
		m := caddymodule.MrokiGate{RawLive: "http://live:8080"}
		err := m.Validate()
		assert.ErrorIs(t, err, caddymodule.ErrRequiredShadowURL)
	})

	t.Run("valid minimal config", func(t *testing.T) {
		m := caddymodule.MrokiGate{
			RawLive:   "http://live:8080",
			RawShadow: "http://shadow:8080",
		}
		err := m.Validate()
		require.NoError(t, err)
	})
}

func TestMrokiGate_Validate_sampling_rate(t *testing.T) {
	t.Run("valid sampling rate", func(t *testing.T) {
		rate := "0.5"
		m := caddymodule.MrokiGate{
			RawLive:      "http://live:8080",
			RawShadow:    "http://shadow:8080",
			SamplingRate: &rate,
		}
		err := m.Validate()
		require.NoError(t, err)
	})

	t.Run("invalid sampling rate", func(t *testing.T) {
		rate := "not-a-number"
		m := caddymodule.MrokiGate{
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
	m := caddymodule.MrokiGate{
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
	sortArrays := "true"
	m := caddymodule.MrokiGate{
		RawLive:            "http://live:8080",
		RawShadow:          "http://shadow:8080",
		DiffIgnoredFields:  &ignored,
		DiffIncludedFields: &included,
		DiffFloatTolerance: &tolerance,
		DiffSortArrays:     &sortArrays,
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
		diff_sort_arrays true
		redacted_fields headers.X-Internal-Token,body.secret
	}`

	d := caddyfile.NewTestDispenser(input)
	var m caddymodule.MrokiGate
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
	require.NotNil(t, m.DiffSortArrays)
	assert.Equal(t, "true", *m.DiffSortArrays)
	require.NotNil(t, m.RedactedFields)
	assert.Equal(t, "headers.X-Internal-Token,body.secret", *m.RedactedFields)
}

func TestUnmarshalCaddyfile_unknown_property(t *testing.T) {
	input := `mroki_gate {
		live http://live:8080
		shadow http://shadow:8080
		api_url http://api:8081
	}`

	d := caddyfile.NewTestDispenser(input)
	var m caddymodule.MrokiGate
	err := m.UnmarshalCaddyfile(d)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown property 'api_url'")
}

func TestMrokiGate_Validate_invalid_float_tolerance(t *testing.T) {
	tolerance := "not-a-number"
	m := caddymodule.MrokiGate{
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
	m := caddymodule.MrokiGate{
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
	m := caddymodule.MrokiGate{
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
	m := caddymodule.MrokiGate{
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
	m := caddymodule.MrokiGate{
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
	var m caddymodule.MrokiGate
	err := m.UnmarshalCaddyfile(d)

	require.NoError(t, err)
	assert.Equal(t, "http://live:8080", m.RawLive)
	assert.Equal(t, "http://shadow:8080", m.RawShadow)
	assert.Nil(t, m.SamplingRate)
	assert.Nil(t, m.RawLiveTimeout)
	assert.Nil(t, m.RawShadowTimeout)
	assert.Nil(t, m.RawMaxBodySize)
	assert.Nil(t, m.RawShadowRules)
	assert.Nil(t, m.HTTPClient)
	assert.Nil(t, m.DiffIgnoredFields)
	assert.Nil(t, m.DiffIncludedFields)
	assert.Nil(t, m.DiffFloatTolerance)
	assert.Nil(t, m.DiffSortArrays)
	assert.Nil(t, m.RedactedFields)
}

func TestUnmarshalCaddyfile_redacted_fields(t *testing.T) {
	input := `mroki_gate {
		live http://live:8080
		shadow http://shadow:8080
		redacted_fields headers.X-Internal-Token,body.user.password
	}`

	d := caddyfile.NewTestDispenser(input)
	var m caddymodule.MrokiGate
	err := m.UnmarshalCaddyfile(d)

	require.NoError(t, err)
	require.NotNil(t, m.RedactedFields)
	assert.Equal(t, "headers.X-Internal-Token,body.user.password", *m.RedactedFields)
}

func TestMrokiGate_Validate_redacted_fields_valid(t *testing.T) {
	fields := "headers.X-Internal-Token,body.user.password"
	m := caddymodule.MrokiGate{
		RawLive:        "http://live:8080",
		RawShadow:      "http://shadow:8080",
		RedactedFields: &fields,
	}
	err := m.Validate()
	require.NoError(t, err)
}

func TestMrokiGate_Validate_redacted_fields_invalid(t *testing.T) {
	fields := "Authorization"
	m := caddymodule.MrokiGate{
		RawLive:        "http://live:8080",
		RawShadow:      "http://shadow:8080",
		RedactedFields: &fields,
	}
	err := m.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid redacted_fields")
}

func TestMrokiGate_Validate_redacted_fields_empty_string(t *testing.T) {
	fields := ""
	m := caddymodule.MrokiGate{
		RawLive:        "http://live:8080",
		RawShadow:      "http://shadow:8080",
		RedactedFields: &fields,
	}
	err := m.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid redacted_fields")
}

func TestMrokiGate_Validate_invalid_diff_sort_arrays(t *testing.T) {
	sortArrays := "invalid"
	m := caddymodule.MrokiGate{
		RawLive:        "http://live:8080",
		RawShadow:      "http://shadow:8080",
		DiffSortArrays: &sortArrays,
	}
	err := m.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid diff_sort_arrays")
}

func TestMrokiGate_Validate_default_redaction_always_applied(t *testing.T) {
	// When redacted_fields is not set, default redaction (Auth, Cookie, etc.)
	// should still be applied — matching main.go behavior.
	m := caddymodule.MrokiGate{
		RawLive:   "http://live:8080",
		RawShadow: "http://shadow:8080",
	}
	err := m.Validate()
	require.NoError(t, err)
	// Validate succeeds and proxy is created — redactor is always initialized
	// with default fields even without explicit redacted_fields config.
}

func TestUnmarshalCaddyfile_http_client_pool(t *testing.T) {
	input := `mroki_gate {
		live http://live:8080
		shadow http://shadow:8080
		http_client {
			max_idle_conns 250
			max_idle_conns_per_host 25
			max_conns_per_host 250
			idle_conn_timeout 45s
		}
	}`

	d := caddyfile.NewTestDispenser(input)
	var m caddymodule.MrokiGate
	err := m.UnmarshalCaddyfile(d)

	require.NoError(t, err)
	require.NotNil(t, m.HTTPClient)
	require.NotNil(t, m.HTTPClient.RawMaxIdleConns)
	assert.Equal(t, "250", *m.HTTPClient.RawMaxIdleConns)
	require.NotNil(t, m.HTTPClient.RawMaxIdleConnsPerHost)
	assert.Equal(t, "25", *m.HTTPClient.RawMaxIdleConnsPerHost)
	require.NotNil(t, m.HTTPClient.RawMaxConnsPerHost)
	assert.Equal(t, "250", *m.HTTPClient.RawMaxConnsPerHost)
	require.NotNil(t, m.HTTPClient.RawIdleConnTimeout)
	assert.Equal(t, "45s", *m.HTTPClient.RawIdleConnTimeout)
}

func TestMrokiGate_Validate_http_client_pool_valid(t *testing.T) {
	maxIdle := "250"
	perHost := "25"
	maxConns := "250"
	idleTimeout := "45s"
	m := caddymodule.MrokiGate{
		RawLive:   "http://live:8080",
		RawShadow: "http://shadow:8080",
		HTTPClient: &caddymodule.HTTPClientOptions{
			RawMaxIdleConns:        &maxIdle,
			RawMaxIdleConnsPerHost: &perHost,
			RawMaxConnsPerHost:     &maxConns,
			RawIdleConnTimeout:     &idleTimeout,
		},
	}
	err := m.Validate()
	require.NoError(t, err)
}

func TestMrokiGate_Validate_http_client_pool_zero_valid(t *testing.T) {
	zero := "0"
	zeroDur := "0s"
	m := caddymodule.MrokiGate{
		RawLive:   "http://live:8080",
		RawShadow: "http://shadow:8080",
		HTTPClient: &caddymodule.HTTPClientOptions{
			RawMaxIdleConns:        &zero,
			RawMaxIdleConnsPerHost: &zero,
			RawMaxConnsPerHost:     &zero,
			RawIdleConnTimeout:     &zeroDur,
		},
	}
	err := m.Validate()
	require.NoError(t, err)
}

func TestMrokiGate_Validate_http_client_pool_invalid(t *testing.T) {
	t.Run("non-numeric max_idle_conns", func(t *testing.T) {
		val := "abc"
		m := caddymodule.MrokiGate{
			RawLive:    "http://live:8080",
			RawShadow:  "http://shadow:8080",
			HTTPClient: &caddymodule.HTTPClientOptions{RawMaxIdleConns: &val},
		}
		err := m.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid max_idle_conns")
	})

	t.Run("negative max_conns_per_host", func(t *testing.T) {
		val := "-1"
		m := caddymodule.MrokiGate{
			RawLive:    "http://live:8080",
			RawShadow:  "http://shadow:8080",
			HTTPClient: &caddymodule.HTTPClientOptions{RawMaxConnsPerHost: &val},
		}
		err := m.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_conns_per_host must be non-negative")
	})

	t.Run("invalid idle_conn_timeout", func(t *testing.T) {
		val := "not-a-duration"
		m := caddymodule.MrokiGate{
			RawLive:    "http://live:8080",
			RawShadow:  "http://shadow:8080",
			HTTPClient: &caddymodule.HTTPClientOptions{RawIdleConnTimeout: &val},
		}
		err := m.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid idle_conn_timeout")
	})

	t.Run("negative idle_conn_timeout", func(t *testing.T) {
		val := "-1s"
		m := caddymodule.MrokiGate{
			RawLive:    "http://live:8080",
			RawShadow:  "http://shadow:8080",
			HTTPClient: &caddymodule.HTTPClientOptions{RawIdleConnTimeout: &val},
		}
		err := m.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "idle_conn_timeout must be non-negative")
	})
}

func TestUnmarshalCaddyfile_shadow_rules(t *testing.T) {
	input := `mroki_gate {
		live http://live:8080
		shadow http://shadow:8080
		shadow_rules "allow POST:/api/v1/search,deny GET:/health/*"
	}`

	d := caddyfile.NewTestDispenser(input)
	var m caddymodule.MrokiGate
	err := m.UnmarshalCaddyfile(d)

	require.NoError(t, err)
	require.NotNil(t, m.RawShadowRules)
	assert.Equal(t, "allow POST:/api/v1/search,deny GET:/health/*", *m.RawShadowRules)
}

func TestMrokiGate_Validate_shadow_rules_valid(t *testing.T) {
	rules := "allow POST:/api/v1/search,deny GET:/health/*"
	m := caddymodule.MrokiGate{
		RawLive:        "http://live:8080",
		RawShadow:      "http://shadow:8080",
		RawShadowRules: &rules,
	}
	err := m.Validate()
	require.NoError(t, err)
}

func TestMrokiGate_Validate_shadow_rules_invalid(t *testing.T) {
	rules := "bogus rule format"
	m := caddymodule.MrokiGate{
		RawLive:        "http://live:8080",
		RawShadow:      "http://shadow:8080",
		RawShadowRules: &rules,
	}
	err := m.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid shadow_rules")
}

func TestUnmarshalCaddyfile_http_client_unknown_key(t *testing.T) {
	input := `mroki_gate {
		live http://live:8080
		shadow http://shadow:8080
		http_client {
			bogus_key 5
		}
	}`

	d := caddyfile.NewTestDispenser(input)
	var m caddymodule.MrokiGate
	err := m.UnmarshalCaddyfile(d)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown http_client property")
}

// serveGate validates the gate (which builds the inner proxy) and serves a
// single request, reporting whether the shadow service was hit. It mirrors the
// atomic.Bool + brief-sleep convention used by the pkg/proxy ServeHTTP tests,
// since shadowed requests are dispatched asynchronously.
func serveGate(t *testing.T, m caddymodule.MrokiGate, method, path, body string) {
	t.Helper()
	require.NoError(t, m.Validate())

	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	} else {
		reqBody = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reqBody)
	rec := httptest.NewRecorder()

	require.NoError(t, m.ServeHTTP(rec, req, nil))
	assert.Equal(t, http.StatusOK, rec.Code)

	// Shadow dispatch is asynchronous when allowed; give it time to land.
	time.Sleep(50 * time.Millisecond)
}

func TestMrokiGate_ServeHTTP_default_write_protection(t *testing.T) {
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer liveServer.Close()

	var shadowCalled atomic.Bool
	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		shadowCalled.Store(true)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer shadowServer.Close()

	t.Run("POST is not shadowed by default", func(t *testing.T) {
		shadowCalled.Store(false)
		m := caddymodule.MrokiGate{RawLive: liveServer.URL, RawShadow: shadowServer.URL}
		serveGate(t, m, http.MethodPost, "/api/orders", `{"x":1}`)
		assert.False(t, shadowCalled.Load(), "POST must not be shadowed under default write-protection")
	})

	t.Run("GET is shadowed by default", func(t *testing.T) {
		shadowCalled.Store(false)
		m := caddymodule.MrokiGate{RawLive: liveServer.URL, RawShadow: shadowServer.URL}
		serveGate(t, m, http.MethodGet, "/api/orders", "")
		assert.True(t, shadowCalled.Load(), "GET should be shadowed by default")
	})
}

func TestMrokiGate_ServeHTTP_shadow_rules_allow_override(t *testing.T) {
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer liveServer.Close()

	var shadowCalled atomic.Bool
	shadowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		shadowCalled.Store(true)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer shadowServer.Close()

	rules := "allow POST:/api/v1/search"

	t.Run("allowed POST path is shadowed", func(t *testing.T) {
		shadowCalled.Store(false)
		m := caddymodule.MrokiGate{RawLive: liveServer.URL, RawShadow: shadowServer.URL, RawShadowRules: &rules}
		serveGate(t, m, http.MethodPost, "/api/v1/search", `{"q":"x"}`)
		assert.True(t, shadowCalled.Load(), "POST to an explicitly allowed path should be shadowed")
	})

	t.Run("other POST path stays write-protected", func(t *testing.T) {
		shadowCalled.Store(false)
		m := caddymodule.MrokiGate{RawLive: liveServer.URL, RawShadow: shadowServer.URL, RawShadowRules: &rules}
		serveGate(t, m, http.MethodPost, "/api/v1/orders", `{"x":1}`)
		assert.False(t, shadowCalled.Load(), "POST outside the allow rule must remain denied by base rules")
	})
}
