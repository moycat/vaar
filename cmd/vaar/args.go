package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/moycat/vaar"
)

type algorithmArg struct {
	value vaar.Algorithm
}

func (arg *algorithmArg) String() string {
	return arg.value.String()
}

func (arg *algorithmArg) Set(s string) error {
	switch strings.ToLower(s) {
	case "":
		arg.value = vaar.NoAlgorithm
	case "gzip":
		arg.value = vaar.GzipAlgorithm
	case "lz4":
		arg.value = vaar.LZ4Algorithm
	default:
		return fmt.Errorf("unknown algorithm '%s'", s)
	}
	return nil
}

type levelArg struct {
	value vaar.Level
}

func (arg *levelArg) String() string {
	return arg.value.String()
}

func (arg *levelArg) Set(s string) error {
	switch strings.ToLower(s) {
	case "", "default":
		arg.value = vaar.DefaultLevel
	case "fastest":
		arg.value = vaar.FastestLevel
	case "fast":
		arg.value = vaar.FastLevel
	case "good":
		arg.value = vaar.GoodLevel
	case "best":
		arg.value = vaar.BestLevel
	default:
		return fmt.Errorf("unknown algorithm level: '%s'", s)
	}
	return nil
}

func parseArgs() *command {
	c := &command{
		algorithm: algorithmArg{value: vaar.NoAlgorithm},
		level:     levelArg{value: vaar.DefaultLevel},
	}
	set := flag.NewFlagSet("Var", flag.ExitOnError)
	set.Var(&c.algorithm, "c", "optional, algorithm algorithm (gzip or lz4)")
	set.Var(&c.level, "l", "[creation] optional, algorithm level (fastest, fast, default, good, best)")
	set.StringVar(&c.extractPath, "d", ".", "[extraction] optional, target path")
	set.IntVar(&c.thread, "t", 4, "[extraction] optional, write thread number")
	set.IntVar(&c.threshold, "s", 512, "[extraction] optional, buffered write threshold in bytes")
	set.IntVar(&c.readAhead, "r", 512, "optional, read ahead number")
	_ = set.Parse(os.Args[1:])
	reportAndExit := func(errMsg string) {
		fmt.Println(errMsg)
		fmt.Println()
		set.Usage()
		os.Exit(2)
	}
	// Parse the operation.
	args := set.Args()
	switch len(args) {
	case 0:
		reportAndExit("Operation is missing: c/create or x/extract")
	case 1:
		reportAndExit("Archive file name is missing.")
	}
	c.archivePath = args[1]
	switch op := strings.ToLower(args[0]); op {
	case "c", "create":
		if len(args) < 3 {
			reportAndExit("Source paths are missing for archive creation.")
		}
		c.operation = "create"
		c.sourcePaths = args[2:]
	case "x", "extract":
		if len(args) > 2 {
			reportAndExit("Too many arguments for extraction.")
		}
		c.operation = "extract"
	default:
		reportAndExit(fmt.Sprintf("Unknown operation %s\nSupported operations: c/create or x/extract", op))
	}
	return c
}
