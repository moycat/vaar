package main

import (
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/moycat/vaar"
)

type command struct {
	operation   string
	archivePath string
	extractPath string
	sourcePaths []string
	// Compression options.
	algorithm algorithmArg
	level     levelArg
	// Parallel options.
	thread    int
	readAhead int
	threshold int
}

func create(cmd *command) {
	log.Println("creating archive", cmd.archivePath, "from", cmd.sourcePaths)
	log.Printf("algorithm: %v, level: %v, read ahead: %d\n", cmd.algorithm.value, cmd.level.value, cmd.readAhead)
	f, err := os.OpenFile(cmd.archivePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		log.Fatalln("failed to create archive file:", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Println("failed to close archive file:", err)
		}
	}()
	ops := []vaar.Option{
		vaar.WithCompression(cmd.algorithm.value),
		vaar.WithLevel(cmd.level.value),
		vaar.WithReadAhead(cmd.readAhead),
	}
	c, err := vaar.NewComposer(f, ops...)
	if err != nil {
		log.Fatalln("failed to create composer:", err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			log.Println("failed to close composer:", err)
		}
	}()
	for _, path := range cmd.sourcePaths {
		if err := c.Add(path, filepath.Dir(path)); err != nil {
			log.Fatalln("failed to add", path, "to tarball:", err)
		}
	}
}

func extract(cmd *command) {
	log.Println("extracting archive", cmd.archivePath, "to", cmd.extractPath)
	log.Printf("algorithm: %v, thread: %d, threshold: %d, read ahead: %d\n", cmd.algorithm.value, cmd.thread, cmd.threshold, cmd.readAhead)
	f, err := os.Open(cmd.archivePath)
	if err != nil {
		log.Fatalln("failed to open archive file:", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Println("failed to close archive file:", err)
		}
	}()
	ops := []vaar.Option{
		vaar.WithCompression(cmd.algorithm.value),
		vaar.WithThread(cmd.thread),
		vaar.WithThreshold(int64(cmd.threshold) << 10),
		vaar.WithReadAhead(cmd.readAhead),
	}
	err = vaar.Resolve(f, cmd.extractPath, ops...)
	if err != nil {
		log.Fatalln("failed to extract tarball:", err)
	}
}

func main() {
	cmd := parseArgs()
	switch cmd.operation {
	case "create":
		create(cmd)
	case "extract":
		extract(cmd)
	}
}

func init() {
	// Set the rlimit to maximum.
	var rlimit syscall.Rlimit
	_ = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	if rlimit.Max > rlimit.Cur {
		rlimit.Cur = rlimit.Max
		_ = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	}
}
