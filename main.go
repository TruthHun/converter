package main

import (
	"fmt"

	"github.com/TruthHun/converter/converter"
)

func main() {
	converter, err := converter.NewConverter("example/book/book.json")
	fmt.Printf("%+v", converter)
	fmt.Println(err)
}
