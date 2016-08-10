package main

import (
    "fmt"
    "crypto/md5"

    "github.com/vazrupe/go-acb/acb"
)

func AcbHashs(acbFile *acb.CriAcbFile) map[string]string {
    table := make(map[string]string)
    for _, cue := range acbFile.Cue {
        file, ok := acbFile.InternalAwb.Files[uint16(cue.CueID)]
        if !ok {
            continue
        }
        
        name := fmt.Sprintf("%s%s", cue.CueName, cue.GetFileExtension())
        hash := fmt.Sprintf("%x", md5.Sum(file.Data))
        
        table[name] = hash
    }
    return table
}

func main() {
    a, err := acb.LoadCriAcbFile("your file path")
    if err != nil {
        panic(err)
    }
    hashs := AcbHashs(a)
    for name, hash := range hashs {
        fmt.Printf("%s: %s\n", name, hash)
    }
}