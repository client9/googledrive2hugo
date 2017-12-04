package main

import (
	"flag"
	"log"

	"github.com/client9/googledrive2hugo"
	"google.golang.org/api/drive/v3"
)

// sample WalkFn
func printer(srv *drive.Service, path string, info *drive.File, err error) error {
	if err != nil {
		log.Printf("Error on %q, %s", path, err)
		return nil
	}
	log.Printf("GOT path=%s, name=%s", path, info.Name)
	return nil
}

func main() {
	rootFlag := flag.String("root", "My Drive", "root to walk")
	flag.Parse()

	srv, err := googledrive2hugo.Setup()
	if err != nil {
		log.Fatalf("unable to auth: %s", err)
	}

	err = googledrive2hugo.Walk(srv, *rootFlag, printer)
	if err != nil {
		log.Fatalf("walk failed: %s", err)
	}
}
