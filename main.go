package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	// "path/filepath"
	"regexp"
	"sync"
	"time"

	// "github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
)

var REGEXP_NARINFO = regexp.MustCompile("/([a-z0-9]{32}).narinfo")
var REGEXP_NAR     = regexp.MustCompile("/nar/([a-z0-9]{52}.nar.*)")

var RICK_REDIRECT = `
<script>window.location.href = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"</script>
`

func init () {
    flag.Parse()
}

type LRUCache struct {
    sync.Mutex
    stateDir string
    upstreams []string
}

func (l *LRUCache) HandleNARInfoRequest(w http.ResponseWriter, r *http.Request) {
    hash := REGEXP_NARINFO.FindStringSubmatch(r.RequestURI)[1]
    filename, err := l.FetchNarinfo(hash)
    if filename == "" {
        w.WriteHeader(404)
        fmt.Fprintf(w, RICK_REDIRECT)
        return
    }
    var f *os.File
    if err == nil {
        defer f.Close()
        f, err = os.Open(filename)
    }
    if err != nil {
        w.WriteHeader(500)
        log.Printf("error when upstreaming NAR request: %s", err.Error())
        return
    }
    w.WriteHeader(200)
    io.Copy(w, f)
    log.Println("narinfo")
}

func (l *LRUCache) HandleNARRequest(w http.ResponseWriter, r *http.Request) {
    hash := REGEXP_NAR.FindStringSubmatch(r.RequestURI)[1]
    filename, err := l.FetchNar(hash)
    if filename == "" {
        w.WriteHeader(404)
        fmt.Fprintf(w, RICK_REDIRECT)
        return
    }
    var f *os.File
    if err == nil {
        defer f.Close()
        f, err = os.Open(filename)
    }
    if err != nil {
        w.WriteHeader(500)
        log.Printf("error when upstreaming NAR request: %s", err.Error())
        return
    }
    w.WriteHeader(200)
    io.Copy(w, f)
    log.Println("nar itself")
}

func (l *LRUCache) HandleLock(w http.ResponseWriter, r *http.Request) {
    l.Lock()
    defer l.Unlock()
    for true {
        _, ok := <-r.Context().Done()
        if !ok {
            break
        }
        time.Sleep(1*time.Second)
    }
}

func (l *LRUCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    log.Printf("request(%s) %s", r.Method, r.RequestURI)
    if r.RequestURI == "/nix-cache-info" {
        fmt.Fprintln(w, "StoreDir: /nix/store")
        fmt.Fprintln(w, "WantMassQuery: 1")
        fmt.Fprintln(w, "Priority: 1")
    } else if r.RequestURI == "/lock" {
        l.HandleLock(w, r)
    } else if REGEXP_NARINFO.MatchString(r.RequestURI) {
        l.HandleNARInfoRequest(w, r)
    } else if REGEXP_NAR.MatchString(r.RequestURI) {
        l.HandleNARRequest(w, r)
    } else {
        fmt.Fprintf(w, "funciona")
    }
}


func (l *LRUCache) Tick() {
    l.Lock()
    defer l.Unlock()
    log.Println("tick")
}

func (l *LRUCache) GetNarDir() string {
    return path.Join(l.stateDir, "nar")
}

func (l *LRUCache) GetNarinfoDir() string {
    return path.Join(l.stateDir, "narinfo")
}

func (l *LRUCache) GetTmpDir() string {
    return path.Join(l.stateDir, "tmp")
}

func (l *LRUCache) FetchNarinfo(hash string) (string, error) {
    narinfoFile := path.Join(l.GetNarinfoDir(), fmt.Sprintf("%s.narinfo", hash))
    if _, err := os.Stat(narinfoFile); errors.Is(err, os.ErrNotExist) {
        tempfile := path.Join(l.GetTmpDir(), uuid.New().String())
        log.Printf("o narinfo %s não existe localmente, baixando...", hash)
        log.Printf("arquivo temporário: %s", tempfile)
        l.Lock()
        defer l.Unlock()
        for _, cacheUrl := range l.upstreams {
            remoteSite := fmt.Sprintf("%s/%s.narinfo", cacheUrl, hash)
            log.Printf("%s", remoteSite)
            res, err := http.Get(remoteSite)
            if err != nil {
                return "", err
            }
            if res.StatusCode == 200 {
                f, err := os.Create(tempfile)
                if err != nil {
                    return "", err
                }
                io.Copy(f, res.Body)
                f.Close()
                err = os.Rename(tempfile, narinfoFile)
                if err != nil {
                    return "", err
                }
            } else {
                continue
            }
        }
    }
    return narinfoFile, nil
}

func (l *LRUCache) FetchNar(hash string) (string, error) {
    return "", nil
}

func NewLRUCache(stateDir string, upstreams ...string) *LRUCache {
    lru := &LRUCache{
        stateDir: stateDir,
        upstreams: upstreams,
    }
    go func() {
        for true {
            lru.Tick()
            time.Sleep(time.Second)
        }
    }()
    os.MkdirAll(lru.GetNarDir(), 0700)
    os.MkdirAll(lru.GetNarinfoDir(), 0700)
    os.MkdirAll(lru.GetTmpDir(), 0700)
    return lru
}


func handleFatalError(err error, context string) {
    if err != nil {
        log.Fatalf("fatal: %s: %s", context, err.Error())
    }
}

func main() {
    var stateDir string
    var listenAddr string
    fmt.Print(`
      _   ___      __    ____  __  __
     / | / (_)  __/ /   / __ \/ / / /
    /  |/ / / |/_/ /   / /_/ / / / / 
   / /|  / />  </ /___/ _, _/ /_/ /  
  /_/ |_/_/_/|_/_____/_/ |_|\____/   

        Nix LRU-based cache


`)
    flag.StringVar(&stateDir, "s", "/tmp/lrucache", "Onde guardar o estado do programa")
    flag.StringVar(&listenAddr, "l", ":8080", "Onde deixar o webserver na escuta")
    flag.Parse()
    handleFatalError(os.MkdirAll(stateDir, 0700), "não foi possível criar a pasta de estado")
    lru := NewLRUCache(stateDir, flag.Args()...)
    log.Printf("Na escuta em '%s'", listenAddr)
    log.Printf("Guardando estado em '%s'", stateDir)
    handleFatalError(http.ListenAndServe(listenAddr, lru), "não foi possível iniciar a aplicação")
}
