# NixLRU

Implementation of a local Nix binary cache in Golang

It is able to run on non Nix based Linux distributions and Windows.

## How to use?

nix-cache-lru [flags] [upstream caches...]

- flags: can be consulted by running `nix-lru --help`
- upstream caches: all the caches, sorted by priority, that will be consulted in case of a cache miss
  - ex: https://cache.nixos.org https://giropops.cachix.org

## Behavior
- This software is used as a substituter/binary cache for one or more clients
- The clients query a narinfo of a nix component
    - The software checks if the narinfo is already in the cache
    - If not, try to download from all the caches until it finds
    - Returns the narinfo data for the user, or a 404 error
- The clients query a nar file referenced by the narinfo
    - The software checks if the nar is in the cache already
    - If not, try to download from each cache until it finds
    - Returns the obtained nar for the user, or a 404 error

Errors in the process are reported and treated with error 500 (internal server error)

Processes that write in the local cache are not paralelized, cache hits are paralellized


## Commands prepared for future use

List nar files by the creation time and size sorting by the creation time

```sh
find /tmp/lrucache -type f -printf "%T@ %s %p\n" | sort -n | grep -v -e '.narinfo$'
```
