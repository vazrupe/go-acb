package main

import (
	"crypto/md5"
	"fmt"

	"github.com/vazrupe/go-acb/acb"
)

func main() {
	a, err := acb.LoadCriAcbFile("your file path")
	if err != nil {
		panic(err)
	}
	for name, data := range a.Files() {
		fmt.Printf("%s: %x\n", name, md5.Sum(data))
	}
}
