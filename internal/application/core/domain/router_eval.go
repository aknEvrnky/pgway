package domain

import (
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

// Resolve iterates over rules in order, returns the first matching rule's target.
// Returns empty string and false if no rule matches.
func (r *Router) Resolve(req *http.Request) (target string, found bool) {
	for _, rule := range r.Rules {
		if rule.Match.Evaluate(req) {
			return rule.Target, true
		}
	}

	return "", false
}

func (m *RouterMatch) Evaluate(r *http.Request) bool {
	// check for not condition
	if m.Not != nil {
		if m.Not.evaluate(r) {
			return false
		}
	}

	if m.Type.IsValid() {
		cond := RouterCondition{
			Type:  m.Type,
			Value: m.Value,
		}

		return cond.evaluate(r)
	}

	if len(m.All) > 0 {
		for _, condition := range m.All {
			if !condition.evaluate(r) {
				return false
			}
		}

		return true
	}

	if len(m.Any) > 0 {
		for _, condition := range m.Any {
			if condition.evaluate(r) {
				return true
			}
		}

		return false
	}

	return false
}

// evaluate evaluates a single condition against the given request.
func (c *RouterCondition) evaluate(r *http.Request) bool {
	switch c.Type {
	case MatchTypeCatchAll:
		return true

	case MatchTypeHost:
		matched, err := filepath.Match(c.Value, r.Host)
		if err != nil {
			return false
		}

		return matched

	case MatchTypeHostSuffix:
		host := r.Host

		// remove port if exists
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}

		// normalize the suffix
		suffix := c.Value
		if !strings.HasPrefix(suffix, ".") {
			suffix = "." + suffix
		}

		return host == strings.TrimPrefix(suffix, ".") || strings.HasSuffix(host, suffix)

	case MatchTypePathPrefix:
		return strings.HasPrefix(r.URL.Path, c.Value)

	case MatchTypePathRegex:
		matched, err := regexp.MatchString(c.Value, r.URL.Path)

		if err != nil {
			return false
		}

		return matched

	case MatchTypeMethod:
		return strings.EqualFold(r.Method, c.Value)

	case MatchTypeHeader:
		// Value format: "HeaderName:HeaderValue"
		// ex: "X-Region:de"
		parts := strings.SplitN(c.Value, ":", 2)
		if len(parts) != 2 {
			return false
		}
		return r.Header.Get(parts[0]) == parts[1]
	}

	return false
}
