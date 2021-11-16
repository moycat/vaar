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
