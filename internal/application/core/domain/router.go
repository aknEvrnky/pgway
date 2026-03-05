package domain

import (
	"errors"
	"fmt"
)

type MatchType string

var (
	ErrNoMatchingRule = errors.New("no matching rule found")
)

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
}

type RouterMatch struct {
	// single condition shorthand
	Type  MatchType `json:"type,omitempty" yaml:"type,omitempty"`
	Value string    `json:"value,omitempty" yaml:"value,omitempty"`

	// composite conditions
	All []RouterCondition `json:"all,omitempty" yaml:"all,omitempty"` // AND
	Any []RouterCondition `json:"any,omitempty" yaml:"any,omitempty"` // OR
	Not *RouterCondition  `json:"not,omitempty" yaml:"not,omitempty"`
}

type RouterRule struct {
	Id     string      `json:"id"`
	Match  RouterMatch `json:"match" yaml:"match"`
	Target string      `json:"target" yaml:"target"`
}

type Router struct {
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

// Validate checks all rules in the router for configuration errors.
// Called at startup, before serving traffic.
func (r *Router) Validate() error {
	for _, rule := range r.Rules {
		if err := rule.validateRule(); err != nil {
			return err
		}
	}

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
