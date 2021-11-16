package vaar

import (
	"strings"

	"github.com/klauspost/compress/gzip"
	"github.com/pierrec/lz4/v4"
	"github.com/pkg/errors"
)

func validateRelPath(path string) error {
	if len(path) == 0 {
		return errors.New("empty path")
	}
	if path[0] == '/' || strings.Contains(path, "/../") || path == ".." ||
		strings.HasPrefix(path, "../") || strings.HasSuffix(path, "/..") {
		return errors.New("forbidden path")
	}
	if strings.ContainsAny(path, "\\\x00") {
		return errors.New("invalid character")
	}
	return nil
}

func getCompressionLevel(algorithm Algorithm, level Level) int64 {
	switch algorithm {
	case LZ4Algorithm:
		switch level {
		case FastestLevel, FastLevel, DefaultLevel:
			return int64(lz4.Fast)
		case GoodLevel:
			return int64(lz4.Level5)
		case BestLevel:
			return int64(lz4.Level9)
		}
	case GzipAlgorithm:
		switch level {
		case FastestLevel:
			return gzip.BestSpeed
		case FastLevel:
			return 3
		case DefaultLevel:
			return gzip.DefaultCompression
		case GoodLevel:
			return 7
		case BestLevel:
			return gzip.BestCompression
		}
	}
	return 0
}
