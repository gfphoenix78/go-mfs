package main

import (
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
)
import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

// parse config file
import "gopkg.in/yaml.v3"

type fileDesc struct {
	name      string
	comp      string
	size_orig int
	size_comp int
	content   []byte
}

var yaml_file string
var output_path string

func init() {
	flag.StringVar(&yaml_file, "yaml", "mfs.yaml", "yaml file defines which files to be compressed in mfs")
	flag.StringVar(&output_path, "opath", "$PWD", "output path")

	flag.Parse()
	if output_path == "$PWD" {
		output_path = os.Getenv("PWD")
	}
}
func prepare() *MFS {
	file, err := os.Open(yaml_file)
	if err != nil {
		log.Fatal(err)
	}

	mfs, err := parseYAML(file)
	if err != nil {
		log.Fatal(err)
	}

	return mfs
}

func build_table(mfs *MFS) map[string]fileDesc {
	var size int
	var content []byte
	tab := make(map[string]fileDesc)
	for _, blk := range mfs.Mfs {
		for _, x := range blk.Entry {
			path := blk.Dir + string(os.PathSeparator)
			if len(x.Path) == 0 {
				path += x.Name
			} else {
				path += x.Path
			}
			if x.Comp == "" {
				x.Comp = "auto"
			}

			file, err := os.Open(path)
			if err != nil {
				log.Fatalln(err)
			}
			size, content = compress_bytes(file, &x.Comp)
			tab[x.Name] = fileDesc{
				name:      x.Name,
				comp:      x.Comp,
				size_orig: size,
				size_comp: len(content),
				content:   content,
			}
			file.Close()
		}
	}
	return tab
}

func main() {
	mfs := prepare()
	tab := build_table(mfs)
	generate_go(tab)

	for _, desc := range tab {
		desc.content = nil
		fmt.Println(desc)
	}
}

func compressHelper(w io.WriteCloser, b *bytes.Buffer, content []byte) []byte {
	_, err := w.Write(content)
	if err != nil {
		log.Fatalln(err)
	}
	w.Close()
	return b.Bytes()
}
func compress_lzw(content []byte) []byte {
	var b bytes.Buffer
	return compressHelper(lzw.NewWriter(&b, lzw.LSB, 8), &b, content)
}
func compress_zlib(content []byte) []byte {
	var b bytes.Buffer
	w, err := zlib.NewWriterLevel(&b, zlib.BestCompression)
	if err != nil {
		panic(err)
	}
	return compressHelper(w, &b, content)
}

func compress_gzip(content []byte) []byte {
	var b bytes.Buffer
	w, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		panic(err)
	}
	return compressHelper(w, &b, content)
}

func compress_auto(content []byte) ([]byte, string) {
	var temp []byte
	res, comp := content, "none"
	if temp = compress_gzip(content); len(temp) < len(res) {
		res = temp
		comp = "gzip"
	}
	if temp = compress_lzw(content); len(temp) < len(res) {
		res = temp
		comp = "lzw"
	}
	if temp = compress_zlib(content); len(temp) < len(res) {
		res = temp
		comp = "zlib"
	}
	return res, comp
}

func compress_bytes(file *os.File, compress *string) (int, []byte) {
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalln(err)
	}
	size_orig := len(content)

	switch *compress {
	case "none":
	case "auto":
		content, *compress = compress_auto(content)
	case "gzip":
		content = compress_gzip(content)
	case "lzw":
		content = compress_lzw(content)
	case "zlib":
		content = compress_zlib(content)
	default:
		log.Printf("unsupported compress tag: %s", *compress)
	}

	return size_orig, content
}

var chart = [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'}

func generate_go(tab map[string]fileDesc) {
	var b bytes.Buffer
	fmt.Fprintln(&b, "package mfs\n")
	fmt.Fprint(&b, "func init() {\n	table = make(map[string]fileDesc)\n")

	for k, v := range tab {
		fmt.Fprintf(&b, "	table[\"%s\"] = fileDesc{\n", k)
		fmt.Fprintf(&b, "		name: \"%s\",\n", v.name)
		fmt.Fprintf(&b, "		comp: \"%s\",\n", v.comp)
		fmt.Fprintf(&b, "		size_orig: %d,\n", v.size_orig)
		fmt.Fprintf(&b, "		size_comp: %d,\n", v.size_comp)
		fmt.Fprint(&b, "		content: []byte{")
		for _, bs := range v.content {
			fmt.Fprintf(&b, `0x%c%c,`, chart[bs>>4], chart[bs&0x0F])
		}
		fmt.Fprintln(&b, "},\n		}")
	}

	fmt.Fprint(&b, "\n}\n")

	output_file := output_path + string(os.PathSeparator) + "mfs_data.go"
	output, err := os.OpenFile(output_file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		log.Fatal(err)
	}
	output.Write(b.Bytes())
	output.Close()
}

type MFS struct {
	Mfs []struct {
		Dir   string
		Entry []Entry
	}
}
type Entry struct {
	Name string
	Path string
	Comp string
}

func parseYAML(r io.Reader) (*MFS, error) {
	d := yaml.NewDecoder(r)
	var m MFS
	err := d.Decode(&m)
	return &m, err
}
