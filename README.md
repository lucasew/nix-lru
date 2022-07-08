# NixLRU

Implementação de cache local para Nix usando Golang

Não tem nada que impede de ser executado em distribuições Linux não NixOS, ou até mesmo sistemas operacionais não Linux.

## Como usar?

nix-cache-lru [flags] [caches upstream...]

- flags: podem ser consultadas através de nix-cache-lru --help
- caches upstream: todos os caches, por ordem de prioridade, que serão consultados em caso de cache miss
  - ex: https://cache.nixos.org https://giropops.cachix.org

## Comportamento
- Este software é utilizado como substituter/cache binário de um ou mais clientes
- Os clientes consultam um narinfo de um componente nix
  - O software checa se esse narinfo já tá em cache
  - Se não estiver, tenta baixar de cada cache até achar
  - Retorna o dado do narinfo obtido para o usuário ou erro 404
- Os clientes baixam o nar referenciado no narinfo
  - O software checa se esse nar já está em cache
  - Se não estiver, tenta baixar de cada cache até achar
  - Retorna o nar obtido para o usuário ou erro 404

Erros no processo são logados e tratados como erro 500 (internal server error)

Processos que gravam no cache local não são paralelizados e apenas um ocorre por vez

Cache hits são paralelizados

## Comandos preparados para uso futuro

Listar arquivos NAR pela data de criação e tamanho ordenando pela data de criação

```sh
find /tmp/lrucache -type f -printf "%T@ %s %p\n" | sort -n | grep -v -e '.narinfo$'
```
