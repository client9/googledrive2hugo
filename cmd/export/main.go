package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/client9/googledrive2hugo"

	"google.golang.org/api/drive/v3"
)

var (
	flagRoot *string
	flagOut  *string
)

func init() {
	flagRoot = flag.String("root", "", "root dir in google drive to use")
	flagOut = flag.String("out", ".", "output directory")
	flag.Parse()
}

// sample WalkFn
func printer(srv *drive.Service, path string, info *drive.File, err error) error {
	if err != nil {
		log.Printf("Error on %q, %s", path, err)
		return nil
	}

	if googledrive2hugo.IsDir(info) {
		outpath := filepath.Dir(filepath.Join(*flagOut, path))
		if outpath == "." || outpath == ".." {
			return nil
		}
		log.Printf("Creating directory %s", outpath)
		err = os.MkdirAll(outpath, 0755)
		if err != nil {
			log.Printf("Unable to make directory %q: %s", filepath.Dir(outpath), err)
			return err
		}
		return nil
	}

	// if not a google doc, skip (or if directory allow Walk to descend)
	if !googledrive2hugo.IsGoogleDoc(info) {
		log.Printf("Skipping %s", path)
		return nil
	}

	log.Printf("GOT path=%s, name=%s", path, info.Name)

	outpath := filepath.Join(*flagOut, path) + ".md"
	outdir := filepath.Dir(outpath)
	if outdir != "." {
		log.Printf("Creating directory %s", outdir)
		err = os.MkdirAll(outdir, 0755)
		if err != nil {
			log.Printf("Unable to make directory %q: %s", outdir, err)
			return err
		}
	}
	rawhtml, err := googledrive2hugo.ExportHTML(srv, info)
	if err != nil {
		log.Printf("WARNING: unable to export %s: %s", path, err)
		return err
	}

	// TODO remove spaces, etc?
	fd, err := os.Create(outpath)
	if err != nil {
		log.Printf("WARNING: unable to create file %s: %s", outpath, err)
		return err
	}
	defer fd.Close()
	log.Printf("Writing: %s", outpath)
	return googledrive2hugo.Convert(rawhtml, info, fd)
}

func main() {
	srv, err := googledrive2hugo.Setup()
	if err != nil {
		log.Fatalf("unable to auth: %s", err)
	}

	err = googledrive2hugo.Walk(srv, *flagRoot, printer)
	if err != nil {
		log.Fatalf("walk failed: %s", err)
	}
}
