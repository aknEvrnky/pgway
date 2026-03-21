package cli

import (
	"bytes"
	"fmt"
	"io"

	"github.com/aknEvrnky/pgway/internal/schema"
	"gopkg.in/yaml.v3"
)

type rawYAMLResource struct {
	Kind     string          `yaml:"kind"`
	Version  string          `yaml:"version"`
	Metadata schema.Metadata `yaml:"metadata"`
	Spec     yaml.Node       `yaml:"spec"`
}

func ParseYAML(data []byte) ([]schema.RawResource, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var resources []schema.RawResource

	for {
		var raw rawYAMLResource

		err := decoder.Decode(&raw)
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("decode yaml document: %w", err)
		}

		if raw.Kind == "" {
			return nil, fmt.Errorf("kind is required")
		}

		if raw.Version == "" {
			return nil, fmt.Errorf("version is required")
		}

		specBytes, err := yaml.Marshal(&raw.Spec)

		if err != nil {
			return nil, fmt.Errorf("marshal spec for %s/%s: %w", raw.Kind, raw.Version, err)
		}

		resources = append(resources, schema.RawResource{
			Kind:     raw.Kind,
			Version:  raw.Version,
			Metadata: raw.Metadata,
			SpecRaw:  specBytes,
		})
	}

	return resources, nil
}
