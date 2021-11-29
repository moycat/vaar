package vaar

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/klauspost/compress/gzip"
	"github.com/pierrec/lz4/v4"
)

const (
	resolveDefaultThread    = 4
	resolveDefaultReadAhead = 1024
	resolveDefaultThreshold = 512 << 10 // 512 KiB
)

// Resolver is the context used when a tarball is being extracted.
// It shouldn't be used directly. Use Resolve instead.
type Resolver struct {
	tr         *tar.Reader
	targetPath string
	thread     int
	readAhead  int
	threshold  int64
	// Compression fields.
	algorithm   Algorithm
	extraCloser io.Closer
	// Runtime status.
	bufferCh  chan *extractOperation // Buffered files are sent here.
	errCh     chan error
	closeCh   chan struct{}
	closeLock sync.Mutex
	wg        sync.WaitGroup
	bufPool   sync.Pool // To reuse files buffers.
}

// extractOperation is an unfinished extract operation, either buffered or synchronous.
type extractOperation struct {
	header *tar.Header
	reader io.Reader
}

// Resolve takes a tarball (optionally compressed) from r and extracts it to targetPath with options.
// The leading slashes of the files are trimmed and path traversal is forbidden.
func Resolve(r io.Reader, targetPath string, options ...Option) error {
	res := &Resolver{
		targetPath: targetPath,
		thread:     resolveDefaultThread,
		readAhead:  resolveDefaultReadAhead,
		threshold:  resolveDefaultThreshold,
	}
	for _, option := range options {
		if err := option(res); err != nil {
			return err
		}
	}
	if err := res.initReader(r); err != nil {
		return err
	}
	res.initRuntime()
	for i := 0; i < res.thread; i++ {
		go res.writeBuffer()
	}
	err := res.readStream()
	res.wg.Wait()
	if err != nil {
		return err
	}
	close(res.errCh)
	return <-res.errCh
}

func (res *Resolver) initReader(r io.Reader) error {
	switch res.algorithm {
	case NoAlgorithm:
	case GzipAlgorithm:
		gr, err := gzip.NewReader(r)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		r = gr
		res.extraCloser = gr
	case LZ4Algorithm:
		lr := lz4.NewReader(r)
		if err := lr.Apply(lz4.ConcurrencyOption(-1)); err != nil {
			return fmt.Errorf("failed to apply lz4 options: %w", err)
		}
		r = lr
	default:
		return ErrUnsupportedAlgorithm
	}
	res.tr = tar.NewReader(r)
	return nil
}

func (res *Resolver) initRuntime() {
	res.bufferCh = make(chan *extractOperation, res.readAhead)
	res.errCh = make(chan error, res.thread)
	res.closeCh = make(chan struct{})
	res.bufPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, res.threshold))
		},
	}
	res.wg.Add(res.thread)
}

// readStream scans the tarball stream and for every file, it buffers and pipes it to a worker,
// or create it synchronously if it's a big regular file exceeding the threshold.
// It exits when the tarball is finished or an error occurs.
func (res *Resolver) readStream() error {
	defer close(res.bufferCh)
	for {
		header, err := res.tr.Next()
		if err != nil {
			if err != io.EOF {
				return fmt.Errorf("failed to read from tar stream: %w", err)
			}
			return nil
		}
		op := &extractOperation{header: header}
		if header.Typeflag == tar.TypeReg {
			// This is a regular file. We need to decide whether to buffer its content and write it asynchronously.
			if header.Size > res.threshold {
				// Too big to buffered. Write it in this thread.
				err := res.writeFile(header, res.tr)
				if err != nil {
					return err
				}
				continue
			}
			buf := res.bufPool.Get().(*bytes.Buffer)
			if _, err := buf.ReadFrom(res.tr); err != nil {
				return fmt.Errorf("failed to read %s from tar stream: %w", header.Name, err)
			}
			op.reader = buf
		}
		select {
		case res.bufferCh <- op:
		case <-res.closeCh:
			return nil
		}
	}
}

// writeBuffer writes all buffered files from the channel.
// It returns when all files are drained or an error occurred.
func (res *Resolver) writeBuffer() {
	defer res.wg.Done()
	for op := range res.bufferCh {
		header, reader := op.header, op.reader
		if err := res.writeFile(header, reader); err != nil {
			res.errCh <- err
			res.cancel()
			return
		}
		if reader != nil {
			buf := reader.(*bytes.Buffer)
			buf.Reset()
			res.bufPool.Put(buf)
		}
	}
}

// writeFile performs the actual write operation, either the synchronous ones and the asynchronous ones.
// Currently, we support directories, regular files, symlinks and hard links.
// TODO: shall we populate UID & GID fields?
func (res *Resolver) writeFile(header *tar.Header, r io.Reader) error {
	name := strings.TrimLeft(header.Name, "/") // Leading slashes are trimmed to make the paths relative.
	if err := validateRelPath(name); err != nil {
		return err
	}
	targetPath := filepath.Join(res.targetPath, name)
	mode := os.FileMode(header.Mode)
	accessTime, modTime := header.AccessTime, header.ModTime
	switch header.Typeflag {
	case tar.TypeDir:
		// Directories are created with the default permission determined by umask.
		if err := os.MkdirAll(targetPath, 0o777); err != nil {
			return err
		}
		// We only chmod here to make sure it's eventually the same with the record in the tarball.
		// All errors from chmod are ignored as some filesystems don't support this.
		// FIXME: if the recorded permission denies write, subsequent writes in this directory fail.
		_ = os.Chmod(targetPath, mode)
	case tar.TypeLink:
		// FIXME: we should try to support real hard links. As the target may not exist yet, we create symlinks instead.
		fallthrough
	case tar.TypeSymlink:
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o777); err != nil {
			return err
		}
		linkTarget := header.Linkname
		if err := os.Symlink(linkTarget, targetPath); err != nil {
			if os.IsExist(err) {
				_ = os.Remove(targetPath)
				err = os.Symlink(linkTarget, targetPath)
			}
			if err != nil {
				return fmt.Errorf("failed to create symlink %s to %s: %w", targetPath, linkTarget, err)
			}
		}
		_ = chmodSymlink(linkTarget, mode)
	case tar.TypeReg:
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o777); err != nil {
			return err
		}
		file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", targetPath, err)
		}
		// FIXME: we should use a copy buffer, but os.File cannot use it.
		if _, err := io.Copy(file, r); err != nil {
			_ = file.Close()
			return fmt.Errorf("failed to write file %s: %w", name, err)
		}
		_ = file.Chmod(mode)
		_ = file.Close()
		// All errors from chtimes are ignored as some filesystems don't support this.
		_ = os.Chtimes(targetPath, accessTime, modTime)
	default:
		return fmt.Errorf("unsupported file type %s", string(header.Typeflag))
	}
	return nil
}

func (res *Resolver) cancel() {
	res.closeLock.Lock()
	select {
	case <-res.closeCh:
	default:
		close(res.closeCh)
	}
	res.closeLock.Unlock()
}
