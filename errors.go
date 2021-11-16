package vaar

import "errors"

var (
	ErrInapplicableOption   = errors.New("option not inapplicable")
	ErrUnknownValue         = errors.New("value is unknown")
	ErrUnsupportedAlgorithm = errors.New("algorithm unsupported")
)
