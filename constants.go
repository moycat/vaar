package vaar

const unknownValue = "unknown"

// Algorithm is the compression algorithm.
type Algorithm uint8

const (
	NoAlgorithm Algorithm = iota
	GzipAlgorithm
	LZ4Algorithm
)

func (c Algorithm) String() string {
	switch c {
	case NoAlgorithm:
		return "none"
	case GzipAlgorithm:
		return "gzip"
	case LZ4Algorithm:
		return "lz4"
	default:
		return unknownValue
	}
}

// Level is the compression level.
type Level uint8

const (
	FastestLevel Level = iota
	FastLevel
	DefaultLevel
	GoodLevel
	BestLevel
)

func (l Level) String() string {
	switch l {
	case FastestLevel:
		return "fastest"
	case FastLevel:
		return "fast"
	case DefaultLevel:
		return "default"
	case GoodLevel:
		return "good"
	case BestLevel:
		return "best"
	default:
		return unknownValue
	}
}
