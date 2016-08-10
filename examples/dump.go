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
	for _, cue := range a.Cue {
		file, ok := a.InternalAwb.Files[uint16(cue.CueID)]
		if !ok {
			continue
		}

		savename := fmt.Sprintf("%s%s", cue.CueName, cue.GetFileExtension())
		fmt.Printf("Write: %s\n", savename)
		f, err := os.Create(savename)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		f.Write(file.Data)
	}
}
