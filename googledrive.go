package googledrive2hugo

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

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
		return err
	}

	if !IsDir(info) {
		return nil
	}

	query := fmt.Sprintf("parents in '%s'", info.Id)
	names, err := srv.Files.List().Q(query).Do()

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
	query := fmt.Sprintf("name='%s'", root)
	r, err := srv.Files.List().Q(query).Do()
	if err != nil {
		return err
	}
	if len(r.Files) != 1 {
		err = fmt.Errorf("0 or more than file for %q found", root)
	}
	info := r.Files[0]
	err = walk(srv, root, info, walkfn)
	if err == filepath.SkipDir {
		return nil
	}
	return err
}

// ExportHTML downloads a Google Doc as HTML
//  Download as Zip and unzip
func ExportHTML(srv *drive.Service, f *drive.File) ([]byte, error) {
	resp, err := srv.Files.Export(f.Id, "text/html").Download()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
