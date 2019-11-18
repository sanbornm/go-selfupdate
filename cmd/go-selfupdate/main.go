package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kr/binarydist"
)

var version, genDir string

type current struct {
	Version string
	Sha256  []byte
}

func generateSha256(path string) ([]byte, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Write(b)
	return h.Sum(nil), nil
}

type gzReader struct {
	z, r io.ReadCloser
}

func (g *gzReader) Read(p []byte) (int, error) {
	return g.z.Read(p)
}

func (g *gzReader) Close() error {
	err := g.z.Close()
	if err != nil {
		return err
	}
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

func getPatch(oldBinary, newBinary io.Reader) (*bytes.Buffer, error) {
	patch := new(bytes.Buffer)
	err := binarydist.Diff(oldBinary, newBinary, patch)
	return patch, err
}

func getPatchFromGzFiles(oldFile, newFile io.ReadCloser) (*bytes.Buffer, error) {
	ar := newGzReader(oldFile)
	defer ar.Close()
	br := newGzReader(newFile)
	defer br.Close()
	return getPatch(ar, br)
}

func compressFile(path string) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	f, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	w.Write(f)
	w.Close() // You must close this first to flush the bytes to the buffer.
	return buf.Bytes(), nil
}

func createUpdate(path string, platform string) {

	hash, err := generateSha256(path)
	if err != nil {
		panic(err)
	}

	c := current{
		Version: version,
		Sha256:  hash,
	}

	b, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(filepath.Join(genDir, platform+".json"), b, 0755)
	if err != nil {
		panic(err)
	}

	os.MkdirAll(filepath.Join(genDir, version), 0755)

	compressedBytes, err := compressFile(path)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(filepath.Join(genDir, version, platform+".gz"), compressedBytes, 0755)

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

		fName := filepath.Join(genDir, file.Name(), platform+".gz")
		old, err := os.Open(fName)
		if err == os.ErrNotExist {
			// Don't have an old release for this os/arch, continue on
			continue
		}

		if err != nil {
			panic(err)
		}

		fName = filepath.Join(genDir, version, platform+".gz")
		newF, err := os.Open(fName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't open %s: error: %s\n", fName, err)
			os.Exit(1)
		}
		patch, err := getPatchFromGzFiles(old, newF)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(filepath.Join(genDir, file.Name(), version, platform), patch.Bytes(), 0755)
		if err != nil {
			panic(err)
		}
	}
}

func printUsage() {
	fmt.Println("")
	fmt.Println("Positional arguments:")
	fmt.Println("\tSingle platform: go-selfupdate myapp 1.2")
	fmt.Println("\tCross platform: go-selfupdate /tmp/mybinares/ 1.2")
}

func createBuildDir() {
	os.MkdirAll(genDir, 0755)
}

func main() {
	outputDirFlag := flag.String("o", "public", "Output directory for writing updates")

	defaultPlatform := runtime.GOOS + "-" + runtime.GOARCH
	goos := os.Getenv("GOOS")
	goarch := os.Getenv("GOARCH")
	if goos != "" && goarch != "" {
		defaultPlatform = goos + "-" + goarch
	}

	platformFlag := flag.String("platform", defaultPlatform,
		"Target platform in the form OS-ARCH. Defaults to running os/arch or the combination of the environment variables GOOS and GOARCH if both are set.")

	flag.Parse()
	if flag.NArg() < 2 {
		flag.Usage()
		printUsage()
		os.Exit(0)
	}

	platform := *platformFlag
	appPath := flag.Arg(0)
	version = flag.Arg(1)
	genDir = *outputDirFlag

	createBuildDir()

	// If dir is given create update for each file
	fi, err := os.Stat(appPath)
	if err != nil {
		panic(err)
	}

	if fi.IsDir() {
		files, err := ioutil.ReadDir(appPath)
		if err == nil {
			for _, file := range files {
				createUpdate(filepath.Join(appPath, file.Name()), file.Name())
			}
			os.Exit(0)
		}
	}

	createUpdate(appPath, platform)
}
