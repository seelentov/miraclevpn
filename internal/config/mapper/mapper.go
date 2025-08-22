// Package mapper provides utilities for mapping between different data structures.
package mapper

import (
	"errors"
	"fmt"
)

var (
	ErrNotImplemented = errors.New("not implemented")
)

func Map(a, b any) (any, error) {
	switch a.(type) {
	default:
		return nil, fmt.Errorf("%w: %T->%T", ErrNotImplemented, a, b)
	}
}
