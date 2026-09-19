package controlplane

import (
	"fmt"
	"strings"
)

const maxNamedDependents = 5

// ResourceDependent names one resource that blocks a delete.
type ResourceDependent struct {
	Type string
	Name string
}

// ResourceInUseError is returned when Delete* would orphan reverse references.
type ResourceInUseError struct {
	ResourceType string
	Name         string
	Dependents   []ResourceDependent
}

func (e *ResourceInUseError) Error() string {
	if e == nil {
		return "resource in use"
	}
	parts := make([]string, 0, maxNamedDependents)
	for i, d := range e.Dependents {
		if i >= maxNamedDependents {
			break
		}
		parts = append(parts, fmt.Sprintf("%s %q", d.Type, d.Name))
	}
	msg := fmt.Sprintf("cannot delete %s %q: still referenced by %s",
		e.ResourceType, e.Name, strings.Join(parts, ", "))
	if extra := len(e.Dependents) - maxNamedDependents; extra > 0 {
		msg = fmt.Sprintf("%s and %d more", msg, extra)
	}
	return msg
}

func newResourceInUse(resourceType, name string, deps []ResourceDependent) *ResourceInUseError {
	return &ResourceInUseError{
		ResourceType: resourceType,
		Name:         name,
		Dependents:   deps,
	}
}

// ResourceMissingRefError is returned when Apply* references a missing target.
type ResourceMissingRefError struct {
	ResourceType string
	Name         string
	MissingType  string
	MissingName  string
}

func (e *ResourceMissingRefError) Error() string {
	if e == nil {
		return "resource missing reference"
	}
	return fmt.Sprintf("cannot apply %s %q: %s %q not found",
		e.ResourceType, e.Name, e.MissingType, e.MissingName)
}

func newResourceMissingRef(resourceType, name, missingType, missingName string) *ResourceMissingRefError {
	return &ResourceMissingRefError{
		ResourceType: resourceType,
		Name:         name,
		MissingType:  missingType,
		MissingName:  missingName,
	}
}
