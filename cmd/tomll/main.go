package main

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

func main() {
	if len(os.Args) >= 2 {
		for _, f := range os.Args[1:] {
			m, e := toml.LoadFile(f)
			if e != nil {
				fmt.Println(e)
			} else {
				fmt.Print(m.ToString())
			}
		}
	}
}
