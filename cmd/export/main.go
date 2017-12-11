package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/client9/googledrive2hugo"

	"google.golang.org/api/drive/v3"

	// for saving intermediate HTML.  The google generated html is
	// compressed
	"github.com/yosssi/gohtml"

	// for converting gdoc filename to something same
	"github.com/gohugoio/hugo/helpers"
)

type nopWriter struct {
	io.Writer
}

func (nopWriter) Close() error { return nil }

// NopWriteCloser returns a WriteCloser with a no-op Close method wrapping
// the provided Reader r.
func NopWriteCloser(w io.Writer) io.WriteCloser {
	return nopWriter{w}
}

var (
	flagRoot     *string
	flagOut      *string
	flagSanitize *bool
	flagSaveTmp  *string
)

func init() {
	flagRoot = flag.String("root", "", "root dir in google drive to use")
	flagOut = flag.String("out", ".", "output directory")
	flagSaveTmp = flag.String("tmp", "", "directory to save intermediate files")
	flagSanitize = flag.Bool("sanitize-filename", true, "sanitize gdoc filename")
	flag.Parse()
}

// sample WalkFn
func printer(srv *drive.Service, path string, info *drive.File, err error) error {

	if err != nil {
		log.Printf("Error on %q, %s", path, err)
		return nil
	}

	if *flagSanitize {
		// PathSpec is a complicated object, but the part we need
		// is simple.  Ideally rip it out of hugo (or make independent function)
		pspec := helpers.PathSpec{}
		path = pspec.URLize(path)
	}

	if googledrive2hugo.IsDir(info) {
		outpath := filepath.Dir(filepath.Join(*flagOut, path))
		if outpath == "." || outpath == ".." {
			return nil
		}
		log.Printf("Creating directory %s", outpath)

		// TODO ADD DRY RUN
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

	rawhtml, err := googledrive2hugo.ExportHTML(srv, info)
	if err != nil {
		log.Printf("WARNING: unable to export %s: %s", path, err)
		return err
	}

	// save raw HTML output if requested
	if *flagSaveTmp != "" {
		htmlpath := filepath.Join(*flagSaveTmp, path) + ".html"
		htmldir := filepath.Dir(htmlpath)
		if htmldir != "." {
			err = os.MkdirAll(htmldir, 0755)
			if err != nil {
				log.Printf("Unable to make %s directory: %s", htmldir, err)
				return err
			}
		}
		log.Printf("Writing HTML: %s", htmlpath)

		nicehtml := gohtml.FormatBytes(rawhtml)
		if err = ioutil.WriteFile(htmlpath, nicehtml, 0644); err != nil {
			return err
		}
	}

	// set up writing to file
	//   dry run by default
	fd := NopWriteCloser(ioutil.Discard)

	// if flagOut is empty, then dry-run only
	if *flagOut != "" {
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
		// markdown time
		fd, err = os.Create(outpath)
		if err != nil {
			log.Printf("WARNING: unable to create file %s: %s", outpath, err)
			return err
		}
		log.Printf("Writing Markdown: %s", outpath)
	}

	defer fd.Close()
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
