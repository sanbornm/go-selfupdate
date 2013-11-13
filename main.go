package main

import (
    "fmt"
    "os"
    "encoding/json"
    "io/ioutil"
    "io"
    "path/filepath"
    "compress/gzip"
    "bytes"
    "crypto/sha256"
    "github.com/kr/binarydist"
    "runtime"
    //"encoding/base64"
)

const (
    plat = runtime.GOOS + "-" + runtime.GOARCH
)

type current struct {
    Version string
    Sha256 []byte
}

func generateSha256(path string) []byte {
    h := sha256.New()
    b, err := ioutil.ReadFile(path)
    if err != nil {
        fmt.Println(err)
    }
    h.Write(b)
    sum := h.Sum(nil)
    return sum
    //return base64.URLEncoding.EncodeToString(sum)
}

type gzReader struct {
        z, r io.ReadCloser
}

func (g *gzReader) Read(p []byte) (int, error) {
        return g.z.Read(p)
}

func (g *gzReader) Close() error {
        g.z.Close()
        return g.r.Close()
}

func newGzReader(r io.ReadCloser) io.ReadCloser {
        var err error
        g := new(gzReader)
        g.r = r
        g.z, err = gzip.NewReader(r)
        if err != nil {
            panic(err)
        }
        return g
}

func main() {
    appPath := os.Args[1]
    version := os.Args[2]
    genDir := "public"
    os.MkdirAll(genDir, 0755)

    c := current{Version: version, Sha256: generateSha256(appPath)}

    b, err := json.MarshalIndent(c, "", "    ")
    if err != nil {
        fmt.Println("error:", err)
    }
    err = ioutil.WriteFile(filepath.Join(genDir, plat + ".json"), b, 0755)
    if err != nil {
        panic(err)
    }

    os.MkdirAll(filepath.Join(genDir, version), 0755)

    var buf bytes.Buffer
    w := gzip.NewWriter(&buf)
    f, err := ioutil.ReadFile(appPath)
    if err != nil {
        panic(err)
    }
    w.Write(f)
    w.Close() // You must close this first to flush the bytes to the buffer.
    err = ioutil.WriteFile(filepath.Join(genDir, version, plat + ".gz"), buf.Bytes(), 0755)

    files, err := ioutil.ReadDir(genDir)
    if err != nil {
        fmt.Println(err)
    }

    for _, file := range files {
        if file.IsDir() == false {
            continue
        }
        if file.Name() == version {
            continue
        }
        os.Mkdir(filepath.Join(genDir, file.Name(), version), 0755)

        fName := filepath.Join(genDir, file.Name(), plat + ".gz")
        old, err := os.Open(fName)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Can't open %s: error: %s\n", fName, err)
            os.Exit(1)
        }

        fName = filepath.Join(genDir, version, plat + ".gz")
        newF, err := os.Open(fName)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Can't open %s: error: %s\n", fName, err)
            os.Exit(1)
        }

        ar := newGzReader(old)
        defer ar.Close()
        br := newGzReader(newF)
        defer br.Close()
        patch := new(bytes.Buffer)
        if err := binarydist.Diff(ar, br, patch); err != nil {
            panic(err)
        }
        ioutil.WriteFile(filepath.Join(genDir, file.Name(), version, plat), patch.Bytes(), 0755)
    }

}
