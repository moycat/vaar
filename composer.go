package vaar

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/gzip"
	"github.com/pierrec/lz4/v4"
)

const (
	composerDefaultReadAhead = 512
	composerDefaultBufSize   = 16 << 20 // 16 MiB
)

// Composer is a tarball creation context.
type Composer struct {
	tw        *tar.Writer
	readAhead int
	bufSize   int
	// Compression fields.
	algorithm   Algorithm
	level       Level
	extraCloser io.Closer
	// Runtime fields.
	buf []byte
}

type addOperation struct {
	header *tar.Header
	reader io.ReadCloser
}

// NewComposer creates a Composer with options, writing the tarball to w.
func NewComposer(w io.Writer, options ...Option) (*Composer, error) {
	c := &Composer{
		readAhead: composerDefaultReadAhead,
		bufSize:   composerDefaultBufSize,
		level:     DefaultLevel,
	}
	// Apply options.
	for _, option := range options {
		if err := option(c); err != nil {
			return nil, err
		}
	}
	// Apply the compression.
	switch c.algorithm {
	case GzipAlgorithm:
		gw, err := gzip.NewWriterLevel(w, int(getCompressionLevel(GzipAlgorithm, c.level)))
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip writer: %w", err)
		}
		w = gw
		c.extraCloser = gw
	case LZ4Algorithm:
		lw := lz4.NewWriter(w)
		if err := lw.Apply(
			lz4.ConcurrencyOption(-1),
			lz4.ChecksumOption(false),
			lz4.CompressionLevelOption(lz4.CompressionLevel(getCompressionLevel(LZ4Algorithm, c.level))),
		); err != nil {
			return nil, fmt.Errorf("failed to apply lz4 options: %w", err)
		}
		w = lw
		c.extraCloser = lw
	case NoAlgorithm:
	default:
		return nil, ErrUnsupportedAlgorithm
	}
	c.tw = tar.NewWriter(w)
	c.buf = make([]byte, c.bufSize)
	return c, nil
}

// Add adds a path to the tarball.
// The path argument is the root path of files being added. If it's a file, only the file is added.
// The paths of all files are trimmed from the prefix filepath.Dir(path).
// The base argument is prepended to the paths of all files using filepath.Join.
// In other words, if you call Add("/a/b/c", "d/e"), all files in /a/b/c are added as d/e/c/... including /a/b/c itself.
func (c *Composer) Add(path, base string) error {
	entry, err := Stat(path)
	if err != nil {
		return err
	}
	if !entry.IsDir() {
		// If the path is a file, just add it and return.
		header, err := getTarHeaderFromEntry(filepath.Join(base, filepath.Base(path)), entry)
		if err != nil {
			return fmt.Errorf("failed to generate header for %s: %w", path, err)
		}
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", path, err)
		}
		defer func() { _ = file.Close() }()
		return c.writeFile(header, file)
	}
	// Start a goroutine to add files.
	opCh := make(chan *addOperation, c.readAhead)
	errCh := make(chan error, 1)
	doneCh := make(chan struct{})
	go c.process(opCh, errCh, doneCh)
	// Walk to do recursive adding.
	adsPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of %s: %w", path, err)
	}
	dirBase := filepath.Dir(adsPath)
	err = Walk(adsPath, func(path string, entry *Entry, r io.ReadCloser) error {
		relPath, err := filepath.Rel(dirBase, path)
		if err != nil {
			return fmt.Errorf("invalid path %s: %w", path, err)
		}
		name := filepath.Join(base, relPath)
		header, err := getTarHeaderFromEntry(name, entry)
		if err != nil {
			return err
		}
		select {
		case opCh <- &addOperation{header: header, reader: r}:
		case err := <-errCh:
			return err
		}
		return nil
	})
	close(opCh)
	<-doneCh
	if err != nil {
		return err
	}
	// Here we return either an error sent by a goroutine, or nil.
	close(errCh)
	return <-errCh
}

func (c *Composer) process(opCh <-chan *addOperation, errCh chan<- error, doneCh chan<- struct{}) {
	defer close(doneCh)
	for op := range opCh {
		header, reader := op.header, op.reader
		err := c.writeFile(header, reader)
		if reader != nil {
			_ = reader.Close()
		}
		if err != nil {
			errCh <- fmt.Errorf("failed to add file %s to tar: %w", header.Name, err)
			break
		}
	}
	// On abnormal conditions，we must drain the channel to close all opened files.
	for op := range opCh {
		if op.reader != nil {
			_ = op.reader.Close()
		}
	}
}

func (c *Composer) writeFile(header *tar.Header, reader io.Reader) error {
	if err := c.tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write header for %s: %w", header.Name, err)
	}
	if header.Typeflag == tar.TypeReg {
		_, err := io.CopyBuffer(c.tw, reader, c.buf)
		if err != nil {
			return fmt.Errorf("failed to write body for %s: %w", header.Name, err)
		}
	}
	return nil
}

// TarWriter returns the underlying tar.Writer.
// It's for manual operations like adding files only. It mustn't be closed.
func (c *Composer) TarWriter() *tar.Writer {
	return c.tw
}

// Close completes the tarball creation. It must be called to flush the buffered bytes.
func (c *Composer) Close() error {
	if c.extraCloser != nil {
		if err := c.extraCloser.Close(); err != nil {
			return fmt.Errorf("failed to close compression writer: %w", err)
		}
	}
	if err := c.tw.Close(); err != nil {
		return fmt.Errorf("failed to close internal tar writer: %w", err)
	}
	return nil
}

// TODO: support hard links.
func getTarHeaderFromEntry(name string, entry *Entry) (*tar.Header, error) {
	header, err := tar.FileInfoHeader(entry, entry.linkname)
	if err != nil {
		return nil, fmt.Errorf("failed to generate header for file %s: %w", name, err)
	}
	// We need to use the PAX header format to support sub-second timestamps.
	header.Format = tar.FormatPAX
	// The filename in fs.FileInfo only has the base name, so we need to modify the name.
	name = strings.TrimLeft(filepath.ToSlash(name), "/")
	if err := validateRelPath(name); err != nil {
		return nil, err
	}
	header.Name = name
	// Fill in the owner fields.
	header.Uid = int(entry.uid)
	header.Gid = int(entry.gid)
	header.Uname = entry.uname
	header.Gname = entry.gname
	return header, nil
}
