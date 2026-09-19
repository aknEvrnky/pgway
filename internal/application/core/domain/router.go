package domain

import (
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

type MatchType string

const (
	MatchTypeHost       MatchType = "host"
	MatchTypeHostSuffix MatchType = "host_suffix"
	MatchTypePathPrefix MatchType = "path_prefix"
	MatchTypePathRegex  MatchType = "path_regex"
	MatchTypeMethod     MatchType = "method"
	MatchTypeHeader     MatchType = "header"
	MatchTypeCatchAll   MatchType = "catch_all"
)

func (m MatchType) IsValid() bool {
	switch m {
	case MatchTypeHost,
		MatchTypeHostSuffix,
		MatchTypePathPrefix,
		MatchTypePathRegex,
		MatchTypeMethod,
		MatchTypeHeader,
		MatchTypeCatchAll:
		return true
	}
	return false
}

type RouterCondition struct {
	Type  MatchType `json:"type" yaml:"type"`
	Value string    `json:"value" yaml:"value"`

	// re holds a pre-compiled path_regex. Not persisted.
	re *regexp.Regexp `json:"-" yaml:"-"`
}

type RouterMatch struct {
	// single condition shorthand
	Type  MatchType `json:"type,omitempty" yaml:"type,omitempty"`
	Value string    `json:"value,omitempty" yaml:"value,omitempty"`

	// composite conditions
	All []RouterCondition `json:"all,omitempty" yaml:"all,omitempty"` // AND
	Any []RouterCondition `json:"any,omitempty" yaml:"any,omitempty"` // OR
	Not *RouterCondition  `json:"not,omitempty" yaml:"not,omitempty"`

	// re holds a pre-compiled shorthand path_regex. Not persisted.
	re *regexp.Regexp `json:"-" yaml:"-"`
}

type RouterRule struct {
	Id     string      `json:"id"`
	Match  RouterMatch `json:"match" yaml:"match"`
	Target string      `json:"target" yaml:"target"`
}

type Router struct {
	Timestamps
	Id          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	Rules       []*RouterRule `json:"rules"`
}

func (m *RouterMatch) HasAll() bool {
	return len(m.All) > 0
}

func (m *RouterMatch) HasAny() bool {
	return len(m.Any) > 0
}

func (m *RouterMatch) HasNot() bool {
	return m.Not != nil
}

// Validate checks all rules in the router for configuration errors and
// pre-compiles path_regex patterns. Called on apply and before serving traffic.
func (r *Router) Validate() error {
	for _, rule := range r.Rules {
		if err := rule.validateRule(); err != nil {
			return err
		}
	}

	return r.Compile()
}

// Compile pre-compiles all path_regex patterns on this router.
// Safe to call after deserialization (re fields are not persisted).
// Does not panic; returns a wrapped error on invalid patterns.
func (r *Router) Compile() error {
	for _, rule := range r.Rules {
		if err := rule.Match.compile(); err != nil {
			return fmt.Errorf("rule %q: %w", rule.Id, err)
		}
	}
	return nil
}

func (m *RouterMatch) compile() error {
	if m.Type == MatchTypePathRegex {
		re, err := regexp.Compile(m.Value)
		if err != nil {
			return fmt.Errorf("path_regex: %w", err)
		}
		m.re = re
	}

	for i := range m.All {
		if err := m.All[i].compile(); err != nil {
			return fmt.Errorf("all: %w", err)
		}
	}
	for i := range m.Any {
		if err := m.Any[i].compile(); err != nil {
			return fmt.Errorf("any: %w", err)
		}
	}
	if m.Not != nil {
		if err := m.Not.compile(); err != nil {
			return fmt.Errorf("not: %w", err)
		}
	}
	return nil
}

func (c *RouterCondition) compile() error {
	if c.Type != MatchTypePathRegex {
		return nil
	}
	re, err := regexp.Compile(c.Value)
	if err != nil {
		return fmt.Errorf("path_regex: %w", err)
	}
	c.re = re
	return nil
}

func (r *RouterRule) validateRule() error {
	m := r.Match

	hasType := m.Type != "" && m.Type.IsValid()

	// we do expect at least one match condition
	if !hasType && !m.HasAll() && !m.HasAny() {
		return fmt.Errorf("rule %q: match must define type, all, or any", r.Id)
	}

	// if user defined a shorthand, it must have a valid value
	// except for match all type
	if hasType && m.Type != MatchTypeCatchAll && m.Value == "" {
		return fmt.Errorf("rule %q: match type %q requires a value", r.Id, m.Type)
	}

	// check condition types
	for _, c := range m.All {
		if !c.Type.IsValid() {
			return fmt.Errorf("rule %q: invalid condition type %q in all", r.Id, c.Type)
		}
	}

	for _, c := range m.Any {
		if !c.Type.IsValid() {
			return fmt.Errorf("rule %q: invalid condition type %q in any", r.Id, c.Type)
		}
	}

	if m.Not != nil && !m.Not.Type.IsValid() {
		return fmt.Errorf("rule %q: invalid condition type %q in not", r.Id, m.Not.Type)
	}

	// check target
	if r.Target == "" {
		return fmt.Errorf("rule %q: target is required", r.Id)
	}

	return nil
}

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
			re:    m.re,
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
		host := hostWithoutPort(r.Host)
		// filepath.Match enables documented glob patterns (e.g. *.example.com).
		matched, err := filepath.Match(c.Value, host)
		if err != nil {
			return false
		}

		return matched

	case MatchTypeHostSuffix:
		host := hostWithoutPort(r.Host)

		// normalize the suffix
		suffix := c.Value
		if !strings.HasPrefix(suffix, ".") {
			suffix = "." + suffix
		}

		return host == strings.TrimPrefix(suffix, ".") || strings.HasSuffix(host, suffix)

	case MatchTypePathPrefix:
		return strings.HasPrefix(r.URL.Path, c.Value)

	case MatchTypePathRegex:
		if c.re == nil {
			return false
		}
		return c.re.MatchString(r.URL.Path)

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

// hostWithoutPort strips a trailing :port from an HTTP Host value.
// Bare hostnames without a port are returned unchanged.
func hostWithoutPort(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return host
}
