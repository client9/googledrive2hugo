package main

import (
	"log"
	"os"

	"github.com/client9/googledrive2hugo"
)

func main() {
	err := googledrive2hugo.Convert(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatalf("Failed: %s", err)
	}
}
