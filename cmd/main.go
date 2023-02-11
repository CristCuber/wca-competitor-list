package main

import (
	"fmt"

	"wca-competitor-list/generater"
)

func main() {
	g := generater.NewGenerater()

	if err := g.GenerateNameList(); err != nil {
		fmt.Println(err)
	}
}
