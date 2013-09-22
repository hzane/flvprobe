package main

import (
	"flag"
	"github.com/heartszhang/flvprobe"
	"os"
)

var (
	input = flag.String(`i`, ``, `flv file path`)
)

func main() {
	flag.Parse()
	if *input == `` {
		flag.PrintDefaults()
		return
	}
	f, e := os.Open(*input)
	if e == nil {
		flvprobe.TraverseFlv(f)
	}
}
