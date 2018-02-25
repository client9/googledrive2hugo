package googledrive2hugo

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"google.golang.org/api/drive/v3"
)

func IsDir(f *drive.File) bool {
	return f.MimeType == "application/vnd.google-apps.folder"
}

func IsGoogleDoc(f *drive.File) bool {
	return f.MimeType == "application/vnd.google-apps.document"
}

type WalkFunc func(srv *drive.Service, path string, info *drive.File, err error) error

func walk(srv *drive.Service, path string, info *drive.File, walkFn WalkFunc) error {
	err := walkFn(srv, path, info, nil)
	if err != nil {
		if IsDir(info) && err != filepath.SkipDir {
			return nil
		}
		return fmt.Errorf("%s: %s", path, err)
	}

	if !IsDir(info) {
		return nil
	}

	query := fmt.Sprintf("parents in '%s'", info.Id)
	names, err := srv.Files.List().Fields("files(id,name,mimeType,createdTime,modifiedTime,description)").Spaces("drive").Corpora("user").Q(query).Do()

	if err != nil {
		return walkFn(srv, path, info, err)
	}

	for _, fileInfo := range names.Files {
		filename := filepath.Join(path, fileInfo.Name)
		if err != nil {
			if err := walkFn(srv, filename, fileInfo, err); err != nil && err != filepath.SkipDir {
				return err
			}
		} else {
			err = walk(srv, filename, fileInfo, walkFn)
			if err != nil {
				if !IsDir(fileInfo) || err != filepath.SkipDir {
					return err
				}
			}
		}
	}
	return nil
}

func Walk(srv *drive.Service, root string, walkfn WalkFunc) error {
	var info *drive.File
	parent := "root"

	// TODO -- this may not be exactly right
	//  but can't find the right thing in golang path module
	parts := strings.Split(root, "/")

	for _, dir := range parts {
		query := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder' and '%s' in parents", dir, parent)
		r, err := srv.Files.List().Spaces("drive").Corpora("user").Q(query).Do()
		if err != nil {
			return err
		}
		if len(r.Files) == 0 {
			return fmt.Errorf("%q not found", root)
		}
		if len(r.Files) > 1 {
			return fmt.Errorf("more than file for %q found", root)
		}
		info = r.Files[0]
		parent = info.Id
	}
	err := walk(srv, "", info, walkfn)
	if err == filepath.SkipDir {
		return nil
	}
	return err
}

// ExportHTML downloads a Google Doc as HTML
//  Download as Zip and unzip
func ExportHTML(srv *drive.Service, f *drive.File) (io.ReadCloser, error) {
	resp, err := srv.Files.Export(f.Id, "text/html").Download()
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// output map is the one used by Hugo
func FileInfoToMeta(fileInfo *drive.File) map[string]interface{} {
	meta := make(map[string]interface{})

	// Google File Metadata:
	// https://godoc.org/google.golang.org/api/drive/v3#File
	//
	// Note: description can only be set via gDrive, not gDocs
	//
	// Hugo Front Matter:
	// https://gohugo.io/content-management/front-matter/
	//
	meta["date"] = fileInfo.CreatedTime
	meta["lastmod"] = fileInfo.ModifiedTime
	if fileInfo.Description != "" {
		meta["description"] = fileInfo.Description
	}

	return meta
}
