package main

import (
	_ "compress/bzip2"
	_ "compress/flate"
	_ "compress/gzip"
	"compress/lzw"
	_ "compress/zlib"
)
import "bytes"
import "flag"
import "fmt"
import "io/ioutil"
import "log"
import "os"

type FileDesc struct {
	name string
	comp string
	size_origin int64
	size_compressed int64
	content []byte
}

type FileConfig struct {
	name string
	compress string
}

var fc = []FileConfig {
	{ "index.html", "none"},
	{ "script.js", "lzw"},
}

var package_name string
var work_dir string

func init() {
	flag.StringVar(&package_name, "package", "ufs", "package to use")

	flag.Parse()
}

func main() {

	package_name = "ufs"
	work_dir = "/Users/hawu/Desktop/aws"
	
	tab := make(map[string][]byte)
	var size int64
	for _, f := range fc {
		path := work_dir + "/" + f.name
		file, err := os.Open(path)
		if err != nil {
			log.Fatalln(err)
		}
		size, tab[f.name] = makeString(file, f.compress)
		fmt.Printf("size = %d\n", size)
		file.Close()
	}

	generate_go(tab)
}

func makeStringLZW(content []byte) []byte {
	var b bytes.Buffer
	w := lzw.NewWriter(&b, lzw.LSB, 8)
	_, err := w.Write(content)
	if err != nil {
		log.Fatalln(err)
	}
	w.Close()
	return b.Bytes()
}
func makeStringNone(content []byte) []byte {
	return content
}
func makeString(file *os.File, compress string) (int64, []byte) {
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalln(err)
	}
	size_orig := int64(len(content))

	switch compress {
	case "none":
	case "lzw":
		content = makeStringLZW(content)
	default:
	}

	return size_orig, content
}
var chart = []byte("0123456789ABCDEF")

func generate_go(tab map[string][]byte) {
	var b bytes.Buffer
	fmt.Fprintln(&b, "package", package_name, "\n")
	fmt.Fprint(&b,
`func init() {
	table = make(map[string][]byte)
`)

	for k, v := range tab {
		fmt.Fprintf(&b, "	table[\"%s\"] = []byte(\"", k)
		for _, bs := range v {
			fmt.Fprintf(&b, `\x%c%c`, chart[bs>>4], chart[bs & 0x0F])
		}
		fmt.Fprintln(&b, "\")")
	}

	fmt.Fprint(&b, "\n}\n")

	output, err := os.OpenFile("output.go", os.O_CREATE | os.O_WRONLY | os.O_TRUNC, 0660)
	if err != nil {
		log.Fatal(err)
	}
	output.Write(b.Bytes())
	output.Close()
}
