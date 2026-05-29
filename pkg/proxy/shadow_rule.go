package proxy

import (
	"fmt"
	"net/http"
	"path"
	"strings"
)

// ShadowRuleAction determines whether a matching request is allowed or denied
// from being shadowed.
type ShadowRuleAction string

const (
	ShadowRuleAllow ShadowRuleAction = "allow"
	ShadowRuleDeny  ShadowRuleAction = "deny"
)

// ShadowRule is a value object representing a single shadow matching rule.
// It combines an action (allow/deny), an HTTP method (or "*" for any), and a
// Caddy-style path pattern.
type ShadowRule struct {
	action ShadowRuleAction
	method string // uppercase HTTP method or "*"
	path   string // Caddy-style path pattern (e.g. "/health/*")
}

// NewShadowRule creates a validated ShadowRule from its components.
func NewShadowRule(action ShadowRuleAction, method, pathPattern string) (ShadowRule, error) {
	if action != ShadowRuleAllow && action != ShadowRuleDeny {
		return ShadowRule{}, fmt.Errorf("invalid shadow rule action: %q, must be \"allow\" or \"deny\"", action)
	}

	method = strings.TrimSpace(method)
	if method == "" {
		return ShadowRule{}, fmt.Errorf("shadow rule method must not be empty")
	}
	method = strings.ToUpper(method)

	pathPattern = strings.TrimSpace(pathPattern)
	if pathPattern == "" {
		return ShadowRule{}, fmt.Errorf("shadow rule path pattern must not be empty")
	}
	// Validate the path pattern up front so malformed patterns are rejected at
	// construction time rather than silently never matching at runtime.
	if err := validatePathPattern(pathPattern); err != nil {
		return ShadowRule{}, fmt.Errorf("invalid shadow rule path pattern %q: %w", pathPattern, err)
	}

	return ShadowRule{action: action, method: method, path: pathPattern}, nil
}

// ParseShadowRule parses a single rule string of the form "ACTION METHOD:path".
// Example: "deny POST:*", "allow GET:/api/v1/*".
func ParseShadowRule(s string) (ShadowRule, error) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return ShadowRule{}, fmt.Errorf("invalid shadow rule format %q: expected \"ACTION METHOD:path\"", s)
	}

	action := ShadowRuleAction(strings.ToLower(parts[0]))

	methodPath := strings.TrimSpace(parts[1])
	colonIdx := strings.Index(methodPath, ":")
	if colonIdx < 0 {
		return ShadowRule{}, fmt.Errorf("invalid shadow rule format %q: expected \"METHOD:path\"", methodPath)
	}

	method := methodPath[:colonIdx]
	pathPattern := methodPath[colonIdx+1:]

	return NewShadowRule(action, method, pathPattern)
}

// ParseShadowRules parses a comma-separated list of shadow rules.
// Example: "deny *:/health/*,allow POST:/api/v1/search,deny POST:*"
func ParseShadowRules(s string) ([]ShadowRule, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	entries := strings.Split(s, ",")
	rules := make([]ShadowRule, 0, len(entries))
	for i, entry := range entries {
		rule, err := ParseShadowRule(entry)
		if err != nil {
			return nil, fmt.Errorf("shadow rule [%d]: %w", i, err)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// BaseShadowRules returns the rules that deny non-idempotent methods (POST,
// PUT, DELETE, PATCH). They are always appended as the final, catch-all entries
// of the rule list so that the write-protection cannot be accidentally dropped
// by configuring custom rules. User-supplied rules are evaluated first and can
// override these per pattern (e.g. "allow POST:/api/v1/search"). GET, HEAD, and
// OPTIONS requests are shadowed by default.
//
// A fresh slice is returned on each call so callers cannot mutate shared state.
func BaseShadowRules() []ShadowRule {
	return []ShadowRule{
		{action: ShadowRuleDeny, method: "POST", path: "*"},
		{action: ShadowRuleDeny, method: "PUT", path: "*"},
		{action: ShadowRuleDeny, method: "DELETE", path: "*"},
		{action: ShadowRuleDeny, method: "PATCH", path: "*"},
	}
}

// Matches reports whether the rule matches the given HTTP method and path.
func (r ShadowRule) Matches(method, reqPath string) bool {
	// Match method
	if r.method != "*" && r.method != strings.ToUpper(method) {
		return false
	}
	return matchPath(r.path, reqPath)
}

// matchPath reports whether a path pattern matches a request path. It mirrors
// the semantics of Caddy's `path` matcher (caddyhttp.MatchPath) so the
// standalone proxy and the caddy-mroki module behave identically.
//
// Matching is case-insensitive and the request path is normalized (doubled
// slashes are merged unless the pattern itself contains "//"). The meaning of
// the "*" wildcard depends on its position:
//
//   - "*"            matches any path.
//   - "/api/*"       trailing "*": prefix match (recursive, crosses "/").
//   - "*.json"       leading "*": suffix match (crosses "/").
//   - "*/admin/*"    leading and trailing "*": substring match.
//   - "/api/*/users" any other "*": single-segment match via path.Match.
func matchPath(pattern, reqPath string) bool {
	pattern = strings.ToLower(pattern)
	reqPath = strings.ToLower(reqPath)

	// Whole-pattern wildcard matches everything.
	if pattern == "*" {
		return true
	}

	// Merge doubled slashes unless the pattern deliberately uses them. This
	// mirrors Caddy and prevents crafted paths from bypassing the matcher.
	reqPath = cleanPath(reqPath, !strings.Contains(pattern, "//"))

	starCount := strings.Count(pattern, "*")

	// Fast substring match: "*x*".
	if starCount == 2 && strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		return strings.Contains(reqPath, pattern[1:len(pattern)-1])
	}

	if starCount == 1 {
		// Fast suffix match: "*x".
		if strings.HasPrefix(pattern, "*") {
			return strings.HasSuffix(reqPath, pattern[1:])
		}
		// Fast prefix match: "x*".
		if strings.HasSuffix(pattern, "*") {
			return strings.HasPrefix(reqPath, pattern[:len(pattern)-1])
		}
	}

	// Single-segment match (via path.Match) for any other "*" placement.
	ok, _ := path.Match(pattern, reqPath)
	return ok
}

// cleanPath normalizes a request path the way Caddy does. When collapseSlashes
// is true, doubled slashes are merged; otherwise empty path segments are
// preserved so that patterns containing "//" can still match.
func cleanPath(p string, collapseSlashes bool) string {
	if collapseSlashes {
		return normalizeSlashes(p)
	}
	// Insert an impossible character between consecutive slashes so path.Clean
	// preserves the empty segment, then strip it back out afterwards.
	const tmpCh = 0xff
	var sb strings.Builder
	for i := 0; i < len(p); i++ {
		if p[i] == '/' && i > 0 && p[i-1] == '/' {
			sb.WriteByte(tmpCh)
		}
		sb.WriteByte(p[i])
	}
	return strings.ReplaceAll(normalizeSlashes(sb.String()), string([]byte{tmpCh}), "")
}

// normalizeSlashes applies path.Clean while preserving a single trailing slash.
func normalizeSlashes(p string) string {
	cleaned := path.Clean(p)
	if cleaned != "/" && strings.HasSuffix(p, "/") {
		cleaned += "/"
	}
	return cleaned
}

// validatePathPattern rejects malformed path patterns, mirroring the dispatch
// in matchPath: fast prefix/suffix/substring patterns are matched literally
// (no path.Match parsing), so only single-segment patterns are validated via
// path.Match.
func validatePathPattern(pattern string) error {
	lower := strings.ToLower(pattern)
	if lower == "*" {
		return nil
	}
	starCount := strings.Count(lower, "*")
	if starCount == 2 && strings.HasPrefix(lower, "*") && strings.HasSuffix(lower, "*") {
		return nil
	}
	if starCount == 1 && (strings.HasPrefix(lower, "*") || strings.HasSuffix(lower, "*")) {
		return nil
	}
	if _, err := path.Match(pattern, ""); err != nil {
		return err
	}
	return nil
}

// Action returns the rule's action.
func (r ShadowRule) Action() ShadowRuleAction { return r.action }

// Method returns the rule's HTTP method.
func (r ShadowRule) Method() string { return r.method }

// Path returns the rule's path pattern.
func (r ShadowRule) Path() string { return r.path }

// String returns the rule in its parseable form: "ACTION METHOD:path".
func (r ShadowRule) String() string {
	return fmt.Sprintf("%s %s:%s", r.action, r.method, r.path)
}

// ShadowRulesCheck evaluates an ordered list of shadow rules against incoming
// requests. Rules are evaluated in definition order; first match wins.
// Unmatched requests are allowed (shadowed).
func ShadowRulesCheck(rules []ShadowRule) CheckFunc {
	return func(r *http.Request) bool {
		for _, rule := range rules {
			if rule.Matches(r.Method, r.URL.Path) {
				return rule.action == ShadowRuleAllow
			}
		}
		// No rule matched — allow shadowing
		return true
	}
}
