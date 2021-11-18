# 🐳 Vaar 📦

**Va**ala **ar**chive is a tar archive tool & library optimized for lots of small files.

Written in Golang, vaar performs operations in parallel & fully utilizes the POSIX APIs to reduce filesystem overheads.

Vaar is capable of tar creation & extraction. It works only on Linux & macOS.

Vaar is in beta. Some bugs are still out there 🙏

## Install

Go 1.16+ is required to compile vaar.

**As a command**

```shell
go install github.com/moycat/vaar/cmd/vaar@latest
```

**As a library**

```go
import "github.com/moycat/vaar"
```

## CLI Usage

Suppose you have `$GOPATH/bin` in your `PATH`.

The common usage to create a tarball is:

```shell
vaar create [-c <algorithm>] [-l <level>] [-r <read_ahead>] <tarball> <file ...>
```

**Arguments:**

- `-c <algorithm>`: Compression algorithm, `lz4` or `gzip`. No compression by default.
- `-l <level>`: Compression level, `fastest`, `fast`, `default`, `good` or `best`.
- `-r <read_ahead>`: Read ahead size, the maximum number of files to be walked and stated ahead. `512` by default.

**Examples:**

- Create a tarball with LZ4 compression: `vaar c -c lz4 archive.tar.lz4 seagrass kombu`
- Create a tarball with a large read ahead size: `vaar c -r 4096 archive.tar shrimps`

The common usage to extract a tarball is:

```shell
vaar extract [-c <algorithm>] [-d <target>] [-s <buffer_threshold>] [-t <thread>] [-r <read_ahead>] <tarball>
```

**Arguments:**

- `-c <algorithm>`: Compression algorithm, `lz4` or `gzip`. No compression by default.
- `-d <target>`: Extraction target path. `.` by default.
- `-s <buffer_threshold>`: The size threshold for a file to be buffered in KiB. `512` by default.
- `-t <thread>`: The number of buffered extraction thread. `4` by default.
- `-r <read_ahead>`: Read ahead size, the maximum number of files to be extracted ahead. `512` by default.

**Examples:**

- Extract a LZ4-compressed tarball to `/tmp`: `vaar x -c lz4 -d /tmp archive.tar.lz4`
- Extract a tarball with high concurrency: `vaar x -s 4096 -t 32 -r 2048 archive.tar`

## Appendix

*Vaal* means *whale* in Estonian, with *Vaala* being its genitive form.

Gigantic [baleen whales](https://en.wikipedia.org/wiki/Baleen_whale) like blue whales eat a large volume of tiny fish and shrimp with extremely high efficiency, by swallowing and filtering tons of water every time they open their mouths.

```
 ________
< Yummy! >
 --------
    \
     \
      \
                    ##        .
              ## ## ##       ==
           ## ## ## ##      ===
       /""""""""""""""""___/ ===
  ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~
       \______ o          __/
        \    \        __/
          \____\______/
```
