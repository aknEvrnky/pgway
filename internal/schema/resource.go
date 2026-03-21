package schema

import "fmt"

type RawResource struct {
	Kind     string   `yaml:"kind" json:"kind"`
	Version  string   `yaml:"version" json:"version"`
	Metadata Metadata `yaml:"metadata" json:"metadata"`
	SpecRaw  []byte   `yaml:"-" json:"-"` // handled by adapter
}

func (r RawResource) Key() string {
	return fmt.Sprintf("%s/%s", r.Kind, r.Version)
}
