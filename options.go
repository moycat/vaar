package vaar

import "errors"

// WithCompression provides the compression algorithm of tar creation and extraction.
func WithCompression(algorithm Algorithm) Option {
	return func(i private) error {
		if algorithm.String() == unknownValue {
			return ErrUnknownValue
		}
		switch i := i.(type) {
		case *Composer:
			i.algorithm = algorithm
		case *Resolver:
			i.algorithm = algorithm
		default:
			return ErrInapplicableOption
		}
		return nil
	}
}

// WithLevel provides the compression level during tar creation.
func WithLevel(level Level) Option {
	return func(i private) error {
		if level.String() == unknownValue {
			return ErrUnknownValue
		}
		c, ok := i.(*Composer)
		if !ok {
			return ErrInapplicableOption
		}
		c.level = level
		return nil
	}
}

// WithThread specifies the worker number during extraction.
func WithThread(n int) Option {
	return func(i private) error {
		if n < 1 {
			return errors.New("thread must be positive")
		}
		r, ok := i.(*Resolver)
		if !ok {
			return ErrInapplicableOption
		}
		r.thread = n
		return nil
	}
}

// WithReadAhead specifies the maximum number of files to be read ahead.
func WithReadAhead(n int) Option {
	return func(i private) error {
		if n < 0 {
			return errors.New("read ahead mustn't be negative")
		}
		switch i := i.(type) {
		case *Composer:
			i.readAhead = n
		case *Resolver:
			i.readAhead = n
		default:
			return ErrInapplicableOption
		}
		return nil
	}
}

// WithThreshold specifies the threshold size in bytes of buffered files during extraction.
func WithThreshold(size int64) Option {
	return func(i private) error {
		if size < 0 {
			return errors.New("threshold mustn't be negative")
		}
		c, ok := i.(*Resolver)
		if !ok {
			return ErrInapplicableOption
		}
		c.threshold = size
		return nil
	}
}

// TODO: WithStrip

// TODO: WithCallback
