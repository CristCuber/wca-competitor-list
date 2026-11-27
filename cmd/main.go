package main

import (
	"flag"
	"fmt"
	"os"

	"wca-competitor-list/generater"
	"wca-competitor-list/web"
)

func main() {
	cli := flag.Bool("cli", false, "ask for the competition id in the terminal and write the PDFs next to the executable")
	addr := flag.String("addr", "127.0.0.1:8080", "address of the local web page")
	noBrowser := flag.Bool("no-browser", false, "do not open the web page in the browser")
	flag.Parse()

	gen := generater.NewGenerater()
	var err error
	if *cli {
		err = gen.GenerateNameList()
	} else {
		err = web.Run(gen, *addr, !*noBrowser)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
