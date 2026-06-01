package proxy_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pedrobarco/mroki/pkg/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxBodySizeCheck(t *testing.T) {
	t.Run("allows requests under limit", func(t *testing.T) {
		check := proxy.MaxBodySizeCheck(100)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 50

		assert.True(t, check(req))
	})

	t.Run("allows requests at exact limit", func(t *testing.T) {
		check := proxy.MaxBodySizeCheck(100)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 100

		assert.True(t, check(req))
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		check := proxy.MaxBodySizeCheck(100)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 101

		assert.False(t, check(req))
	})

	t.Run("blocks chunked encoding requests", func(t *testing.T) {
		check := proxy.MaxBodySizeCheck(100)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = -1 // Chunked encoding

		assert.False(t, check(req))
	})

	t.Run("allows all requests when limit is 0", func(t *testing.T) {
		check := proxy.MaxBodySizeCheck(0)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 999999

		assert.True(t, check(req))
	})

	t.Run("allows all requests when limit is negative", func(t *testing.T) {
		check := proxy.MaxBodySizeCheck(-1)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 999999

		assert.True(t, check(req))
	})

	t.Run("allows chunked when limit is 0", func(t *testing.T) {
		check := proxy.MaxBodySizeCheck(0)
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = -1

		assert.True(t, check(req))
	})
}

func TestSamplingRateCheck(t *testing.T) {
	t.Run("allows all requests when rate is nil", func(t *testing.T) {
		check := proxy.SamplingRateCheck(nil)
		req := httptest.NewRequest("GET", "/test", nil)

		// Should always return true
		for i := 0; i < 100; i++ {
			assert.True(t, check(req))
		}
	})

	t.Run("respects sampling rate", func(t *testing.T) {
		rate, _ := proxy.NewSamplingRate(0.5)
		check := proxy.SamplingRateCheck(rate)
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
		rate, _ := proxy.NewSamplingRate(0.0)
		check := proxy.SamplingRateCheck(rate)
		req := httptest.NewRequest("GET", "/test", nil)

		// Should never sample
		for i := 0; i < 100; i++ {
			assert.False(t, check(req))
		}
	})

	t.Run("sampling rate 1 allows all requests", func(t *testing.T) {
		rate, _ := proxy.NewSamplingRate(1.0)
		check := proxy.SamplingRateCheck(rate)
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
		rate, _ := proxy.NewSamplingRate(1.0) // Always sample
		bodySizeCheck := proxy.MaxBodySizeCheck(100)
		samplingCheck := proxy.SamplingRateCheck(rate)

		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 50

		// Both should pass
		assert.True(t, bodySizeCheck(req))
		assert.True(t, samplingCheck(req))
	})
}

func TestNewShadowRule(t *testing.T) {
	t.Run("valid rule", func(t *testing.T) {
		rule, err := proxy.NewShadowRule(proxy.ShadowRuleDeny, "POST", "/api/*")
		require.NoError(t, err)
		assert.Equal(t, proxy.ShadowRuleDeny, rule.Action())
		assert.Equal(t, "POST", rule.Method())
		assert.Equal(t, "/api/*", rule.Path())
	})

	t.Run("uppercases method", func(t *testing.T) {
		rule, err := proxy.NewShadowRule(proxy.ShadowRuleAllow, "get", "/test")
		require.NoError(t, err)
		assert.Equal(t, "GET", rule.Method())
	})

	t.Run("rejects invalid action", func(t *testing.T) {
		_, err := proxy.NewShadowRule("block", "GET", "/test")
		assert.Error(t, err)
	})

	t.Run("rejects empty method", func(t *testing.T) {
		_, err := proxy.NewShadowRule(proxy.ShadowRuleDeny, "", "/test")
		assert.Error(t, err)
	})

	t.Run("rejects empty path", func(t *testing.T) {
		_, err := proxy.NewShadowRule(proxy.ShadowRuleDeny, "GET", "")
		assert.Error(t, err)
	})

	t.Run("rejects malformed path pattern", func(t *testing.T) {
		_, err := proxy.NewShadowRule(proxy.ShadowRuleDeny, "GET", "/api/[")
		assert.Error(t, err)
	})
}

func TestParseShadowRule(t *testing.T) {
	t.Run("parses deny rule", func(t *testing.T) {
		rule, err := proxy.ParseShadowRule("deny POST:*")
		require.NoError(t, err)
		assert.Equal(t, proxy.ShadowRuleDeny, rule.Action())
		assert.Equal(t, "POST", rule.Method())
		assert.Equal(t, "*", rule.Path())
	})

	t.Run("parses allow rule with path", func(t *testing.T) {
		rule, err := proxy.ParseShadowRule("allow GET:/api/v1/*")
		require.NoError(t, err)
		assert.Equal(t, proxy.ShadowRuleAllow, rule.Action())
		assert.Equal(t, "GET", rule.Method())
		assert.Equal(t, "/api/v1/*", rule.Path())
	})

	t.Run("parses wildcard method", func(t *testing.T) {
		rule, err := proxy.ParseShadowRule("deny *:/health/*")
		require.NoError(t, err)
		assert.Equal(t, "*", rule.Method())
		assert.Equal(t, "/health/*", rule.Path())
	})

	t.Run("rejects missing colon", func(t *testing.T) {
		_, err := proxy.ParseShadowRule("deny POST/test")
		assert.Error(t, err)
	})

	t.Run("rejects single token", func(t *testing.T) {
		_, err := proxy.ParseShadowRule("deny")
		assert.Error(t, err)
	})
}

func TestParseShadowRules(t *testing.T) {
	t.Run("parses comma-separated rules", func(t *testing.T) {
		rules, err := proxy.ParseShadowRules("deny *:/health/*,allow POST:/api/v1/search,deny POST:*")
		require.NoError(t, err)
		assert.Len(t, rules, 3)
		assert.Equal(t, proxy.ShadowRuleDeny, rules[0].Action())
		assert.Equal(t, proxy.ShadowRuleAllow, rules[1].Action())
		assert.Equal(t, proxy.ShadowRuleDeny, rules[2].Action())
	})

	t.Run("returns nil for empty string", func(t *testing.T) {
		rules, err := proxy.ParseShadowRules("")
		require.NoError(t, err)
		assert.Nil(t, rules)
	})

	t.Run("returns error on invalid entry", func(t *testing.T) {
		_, err := proxy.ParseShadowRules("deny POST:*,invalid")
		assert.Error(t, err)
	})
}

func TestShadowRule_Matches(t *testing.T) {
	t.Run("exact method and path", func(t *testing.T) {
		rule, _ := proxy.NewShadowRule(proxy.ShadowRuleDeny, "POST", "/test")
		assert.True(t, rule.Matches("POST", "/test"))
		assert.False(t, rule.Matches("GET", "/test"))
		assert.False(t, rule.Matches("POST", "/other"))
	})

	t.Run("wildcard method", func(t *testing.T) {
		rule, _ := proxy.NewShadowRule(proxy.ShadowRuleDeny, "*", "/health")
		assert.True(t, rule.Matches("GET", "/health"))
		assert.True(t, rule.Matches("POST", "/health"))
	})

	t.Run("trailing star is a recursive prefix match", func(t *testing.T) {
		rule, _ := proxy.NewShadowRule(proxy.ShadowRuleDeny, "*", "/health/*")
		assert.True(t, rule.Matches("GET", "/health/ready"))
		assert.True(t, rule.Matches("GET", "/health/ready/deep"))
		assert.False(t, rule.Matches("GET", "/health"))
		assert.False(t, rule.Matches("GET", "/api/health/ready"))
	})

	t.Run("leading star is a suffix match", func(t *testing.T) {
		rule, _ := proxy.NewShadowRule(proxy.ShadowRuleDeny, "*", "*.json")
		assert.True(t, rule.Matches("GET", "/api/data.json"))
		assert.True(t, rule.Matches("GET", "/a/b/c.json"))
		assert.False(t, rule.Matches("GET", "/api/data.xml"))
	})

	t.Run("leading and trailing star is a substring match", func(t *testing.T) {
		rule, _ := proxy.NewShadowRule(proxy.ShadowRuleDeny, "*", "*/admin/*")
		assert.True(t, rule.Matches("GET", "/api/admin/users"))
		assert.True(t, rule.Matches("GET", "/v1/admin/x/y"))
		assert.False(t, rule.Matches("GET", "/api/public/users"))
	})

	t.Run("mid-pattern star matches a single segment", func(t *testing.T) {
		rule, _ := proxy.NewShadowRule(proxy.ShadowRuleDeny, "*", "/gates/*/requests/*/details")
		assert.True(t, rule.Matches("GET", "/gates/abc/requests/def/details"))
		assert.False(t, rule.Matches("GET", "/gates/a/b/requests/c/details"))
		assert.False(t, rule.Matches("GET", "/gates/abc/requests/def/other"))
	})

	t.Run("matching is case-insensitive", func(t *testing.T) {
		rule, _ := proxy.NewShadowRule(proxy.ShadowRuleDeny, "*", "/Health/*")
		assert.True(t, rule.Matches("GET", "/health/READY"))
	})

	t.Run("doubled slashes are merged", func(t *testing.T) {
		rule, _ := proxy.NewShadowRule(proxy.ShadowRuleDeny, "*", "/health/*")
		assert.True(t, rule.Matches("GET", "/health//ready"))
	})

	t.Run("doubled slashes preserved when the pattern uses them", func(t *testing.T) {
		// A pattern that deliberately contains "//" disables slash-merging, so
		// the empty segment in the request path is preserved for matching.
		rule, _ := proxy.NewShadowRule(proxy.ShadowRuleDeny, "*", "/health//*")
		assert.True(t, rule.Matches("GET", "/health//ready"))
		assert.False(t, rule.Matches("GET", "/health/ready"))
	})

	t.Run("bare star matches everything", func(t *testing.T) {
		rule, _ := proxy.NewShadowRule(proxy.ShadowRuleDeny, "POST", "*")
		assert.True(t, rule.Matches("POST", "anything"))
		assert.True(t, rule.Matches("POST", "/deep/path"))
		assert.True(t, rule.Matches("POST", "/"))
	})
}

func TestShadowRulesCheck(t *testing.T) {
	t.Run("first match wins — deny", func(t *testing.T) {
		rules := []proxy.ShadowRule{
			mustRule(t, proxy.ShadowRuleDeny, "*", "/health/*"),
			mustRule(t, proxy.ShadowRuleAllow, "GET", "/health/ready"),
		}
		check := proxy.ShadowRulesCheck(rules)
		req := httptest.NewRequest("GET", "/health/ready", nil)
		// First rule matches → deny
		assert.False(t, check(req))
	})

	t.Run("first match wins — allow", func(t *testing.T) {
		rules := []proxy.ShadowRule{
			mustRule(t, proxy.ShadowRuleAllow, "POST", "/api/v1/search"),
			mustRule(t, proxy.ShadowRuleDeny, "POST", "*"),
		}
		check := proxy.ShadowRulesCheck(rules)
		req := httptest.NewRequest("POST", "/api/v1/search", nil)
		assert.True(t, check(req))
	})

	t.Run("unmatched requests are allowed", func(t *testing.T) {
		rules := []proxy.ShadowRule{
			mustRule(t, proxy.ShadowRuleDeny, "POST", "*"),
		}
		check := proxy.ShadowRulesCheck(rules)
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		assert.True(t, check(req))
	})

	t.Run("base rules deny non-idempotent methods", func(t *testing.T) {
		check := proxy.ShadowRulesCheck(proxy.BaseShadowRules())

		for _, method := range []string{"POST", "PUT", "DELETE", "PATCH"} {
			req := httptest.NewRequest(method, "/api/v1/users", nil)
			assert.False(t, check(req), "expected %s to be denied", method)
		}
	})

	t.Run("base rules allow idempotent methods", func(t *testing.T) {
		check := proxy.ShadowRulesCheck(proxy.BaseShadowRules())

		for _, method := range []string{"GET", "HEAD", "OPTIONS"} {
			req := httptest.NewRequest(method, "/api/v1/users", nil)
			assert.True(t, check(req), "expected %s to be allowed", method)
		}
	})

	t.Run("user rule overrides base catch-all", func(t *testing.T) {
		// User rules first, base rules appended last (as main.go wires them).
		rules := append([]proxy.ShadowRule{
			mustRule(t, proxy.ShadowRuleAllow, "POST", "/api/v1/search"),
		}, proxy.BaseShadowRules()...)
		check := proxy.ShadowRulesCheck(rules)

		// Overridden route is shadowed despite the base POST deny.
		assert.True(t, check(httptest.NewRequest("POST", "/api/v1/search", nil)))
		// Other POSTs still fall through to the base deny.
		assert.False(t, check(httptest.NewRequest("POST", "/api/v1/users", nil)))
	})

	t.Run("empty rules allow everything", func(t *testing.T) {
		check := proxy.ShadowRulesCheck(nil)
		req := httptest.NewRequest("POST", "/api/v1/users", nil)
		assert.True(t, check(req))
	})
}

func TestShadowRule_String(t *testing.T) {
	rule, _ := proxy.NewShadowRule(proxy.ShadowRuleDeny, "POST", "*")
	assert.Equal(t, "deny POST:*", rule.String())
}

func mustRule(t *testing.T, action proxy.ShadowRuleAction, method, path string) proxy.ShadowRule {
	t.Helper()
	rule, err := proxy.NewShadowRule(action, method, path)
	require.NoError(t, err)
	return rule
}
