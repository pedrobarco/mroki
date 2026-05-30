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
// glob-style path pattern.
type ShadowRule struct {
	action ShadowRuleAction
	method string // uppercase HTTP method or "*"
	path   string // glob pattern (e.g. "/health/*")
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

// DefaultShadowRules denies non-idempotent methods (POST, PUT, DELETE, PATCH).
// GET, HEAD, and OPTIONS requests are shadowed by default.
var DefaultShadowRules = []ShadowRule{
	{action: ShadowRuleDeny, method: "POST", path: "*"},
	{action: ShadowRuleDeny, method: "PUT", path: "*"},
	{action: ShadowRuleDeny, method: "DELETE", path: "*"},
	{action: ShadowRuleDeny, method: "PATCH", path: "*"},
}

// Matches reports whether the rule matches the given HTTP method and path.
func (r ShadowRule) Matches(method, reqPath string) bool {
	// Match method
	if r.method != "*" && r.method != strings.ToUpper(method) {
		return false
	}

	// Match path — bare "*" matches everything; otherwise use glob semantics
	if r.path == "*" {
		return true
	}
	matched, _ := path.Match(r.path, reqPath)
	return matched
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
