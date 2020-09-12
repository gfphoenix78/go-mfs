package mfs

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
)

type fileDesc struct {
	name      string
	comp      string
	size_orig int
	size_comp int
	content   []byte
}

// table should be initialized in the generated code
var table map[string]fileDesc

type wrapClose struct {
	*bytes.Reader
}

func (c wrapClose) Close() error {
	return nil
}

func Open(name string) (io.ReadCloser, error) {
	desc, ex := table[name]
	if !ex {
		return os.Open(name)
	}
	if desc.size_comp != len(desc.content) {
		panic("unmatched size")
	}

	br := bytes.NewReader(desc.content)
	switch desc.comp {
	case "none":
		return wrapClose{br}, nil
	case "gzip":
		return gzip.NewReader(br)
	case "lzw":
		return lzw.NewReader(br, lzw.LSB, 8), nil
	case "zlib":
		return zlib.NewReader(br)
	default:
	}
	desc.content = nil
	panic(desc)
}

func List() []string {
	keys := make([]string, 0, len(table))
	for key, _ := range table {
		keys = append(keys, key)
	}
	return keys
}

func Free(name string) {
	delete(table, name)
}

var NoSuchFile = errors.New("No such file")

func LookupFromTar(r io.Reader, name string) ([]byte, error) {
	t := tar.NewReader(r)
	var h *tar.Header
	var err error
	for h, err = t.Next(); err == nil; h, err = t.Next() {
		if h.Name == name {
			b := make([]byte, h.Size)
			n, err := t.Read(b)
			if int64(n) != h.Size {
				log.Printf("expected %d bytes, read %d bytes", h.Size, n)
			}
			return b, err
		}
	}
	if err == nil {
		err = NoSuchFile
	}
	return nil, err
}

// zip Reader is newly initialized
func LookupFromZip(r zip.Reader, name string) ([]byte, error) {
	for _, f := range r.File {
		if f.Name == name {
			target, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer target.Close()
			return ioutil.ReadAll(target)
		}
	}
	return nil, NoSuchFile
}
