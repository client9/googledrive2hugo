package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/client9/googledrive2hugo"
	"github.com/client9/ilog"
	"github.com/client9/ilog/stdlib/adapter"
	"google.golang.org/api/drive/v3"

	// for saving intermediate HTML.  The google generated html is
	// compressed
	"github.com/client9/htmlfmt"

	// for converting gdoc filename to something same
	"github.com/gohugoio/hugo/helpers"
)

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
func walker(c googledrive2hugo.Converter, logger ilog.Logger) googledrive2hugo.WalkFunc {
	return func(srv *drive.Service, path string, info *drive.File, err error) error {
		origpath := path
		if err != nil {
			logger.Debug("walk error", "path", path, "err", err)
			return nil
		}

		if *flagSanitize {
			// PathSpec is a complicated object, but the part we need
			// is simple.  Ideally rip it out of hugo
			// (or make independent function)
			pspec := helpers.PathSpec{}
			path = pspec.URLize(path)
		}

		if googledrive2hugo.IsDir(info) {
			outpath := filepath.Dir(filepath.Join(*flagOut, path))
			if outpath == "." || outpath == ".." {
				return nil
			}
			// TODO ADD DRY RUN
			err = os.MkdirAll(outpath, 0755)
			if err != nil {
				return err
			}
			return nil
		}

		// if not a google doc, skip (or if directory allow Walk to descend)
		if !googledrive2hugo.IsGoogleDoc(info) {
			logger.Debug("skipping non google doc", "name", path)
			return nil
		}
		logger.Debug("reading", "path", origpath)
		rawhtml, err := googledrive2hugo.ExportHTML(srv, info)
		if err != nil {
			return err
		}

		// save raw HTML output if requested
		if *flagSaveTmp != "" {
			htmlpath := filepath.Join(*flagSaveTmp, path) + ".html"
			htmldir := filepath.Dir(htmlpath)
			if htmldir != "." {
				if err = os.MkdirAll(htmldir, 0755); err != nil {
					return err
				}
			}
			rawhtml := htmlfmt.FormatBytes(rawhtml, "", "  ")
			if err = ioutil.WriteFile(htmlpath, rawhtml, 0644); err != nil {
				return err
			}
		}
		fileMeta := googledrive2hugo.FileInfoToMeta(info)
		out, err := c.ToHTML(rawhtml, fileMeta)
		if err != nil {
			return err
		}
		if *flagOut == "" {
			return nil
		}
		outpath := filepath.Join(*flagOut, path) + ".html"
		outdir := filepath.Dir(outpath)
		if outdir != "." {
			if err = os.MkdirAll(outdir, 0755); err != nil {
				return err
			}
		}
		logger.Debug("writing html", "path", outpath)
		if err = ioutil.WriteFile(outpath, out, 0644); err != nil {
			return err
		}
		return nil
	}
}

func main() {
	stdlog := log.New(os.Stderr, "", 0)
	logger := adapter.New(stdlog)
	convert := googledrive2hugo.Converter{
		Logger: logger,
	}

	srv, err := googledrive2hugo.Setup()
	if err != nil {
		logger.Error("unable to auth", "err", err)
		os.Exit(1)
	}

	err = googledrive2hugo.Walk(srv, *flagRoot, walker(convert, logger))
	if err != nil {
		logger.Error("walk failed", "err", err)
		os.Exit(1)
	}
}
