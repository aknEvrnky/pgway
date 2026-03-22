package v1

import "fmt"

type RouterSpecV1 struct {
	Title       string     `yaml:"title,omitempty" json:"title,omitempty"`
	Description string     `yaml:"description,omitempty" json:"description,omitempty"`
	Rules       []RuleSpec `yaml:"rules" json:"rules"`
}

type RuleSpec struct {
	Id     string    `yaml:"id" json:"id"`
	Match  MatchSpec `yaml:"match" json:"match"`
	Target string    `yaml:"target" json:"target"`
}

type MatchSpec struct {
	Type  string          `yaml:"type,omitempty" json:"type,omitempty"`
	Value string          `yaml:"value,omitempty" json:"value,omitempty"`
	All   []ConditionSpec `yaml:"all,omitempty" json:"all,omitempty"`
	Any   []ConditionSpec `yaml:"any,omitempty" json:"any,omitempty"`
	Not   *ConditionSpec  `yaml:"not,omitempty" json:"not,omitempty"`
}

type ConditionSpec struct {
	Type  string `yaml:"type" json:"type"`
	Value string `yaml:"value" json:"value"`
}

func (s RouterSpecV1) Validate() error {
	if len(s.Rules) == 0 {
		return fmt.Errorf("spec.rules is required")
	}

	for i, rule := range s.Rules {
		if rule.Id == "" {
			return fmt.Errorf("spec.rules[%d].id is required", i)
		}
		if rule.Target == "" {
			return fmt.Errorf("spec.rules[%d].target is required", i)
		}

		m := rule.Match
		hasType := m.Type != ""
		hasAll := len(m.All) > 0
		hasAny := len(m.Any) > 0

		if !hasType && !hasAll && !hasAny {
			return fmt.Errorf("spec.rules[%d].match must define type, all, or any", i)
		}

		if hasType && m.Type != "catch_all" && m.Value == "" {
			return fmt.Errorf("spec.rules[%d].match type %q requires a value", i, m.Type)
		}
	}

	return nil
}
