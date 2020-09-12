package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"mfs"
	"os"
)

var work_dir string
var test_name string

func init() {
	flag.StringVar(&work_dir, "cwd", "$PWD", "directory")
	flag.StringVar(&test_name, "test", "", "test name")
	flag.Parse()

	if work_dir == "$PWD" {
		work_dir = os.Getenv("PWD")
	}
}

func run_test(test_name string) {
	rc, err := mfs.Open(test_name)
	if err != nil {
		panic(err)
	}
	defer rc.Close()
	content, err := ioutil.ReadAll(rc)
	if err != nil {
		panic(err)
	}
	file, err := os.Open(work_dir + string(os.PathSeparator) + test_name)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	orig, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	e := bytes.Compare(content, orig)
	if e != 0 {
		panic("fail")
	}
	fmt.Printf("SUCCESS: %s\n", test_name)
}

func main() {
	var keys []string
	if len(test_name) > 0 {
		keys = []string{test_name}
	} else {
		keys = mfs.List()
	}
	for _, key := range keys {
		run_test(key)
	}
}
