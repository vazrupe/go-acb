package main

import (
	"fmt"
	"os"

	"github.com/vazrupe/go-acb/acb"
)

func main() {
	a, err := acb.LoadCriAcbFile("your file path")
	if err != nil {
		panic(err)
	}
	for name, data := range a.Files() {
		fmt.Printf("Write: %s\n", name)

		f, err := os.Create(name)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		f.Write(data)
	}
}
