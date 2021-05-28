package compress

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"

	"github.com/docker/docker/pkg/system"

	"io"
	"io/ioutil"

	"os"
	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/utils"
)

func validatePath(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("dir %s must be absolute path", path)
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("dir %s does not exist, err: %s", path, err)
	}
	return nil
}

// Compress
// src is the dir or single file to tar
// not contain the dir
// newFolder is a folder for tar file
func Compress(targetFile *os.File, paths ...string) (file *os.File, err error) {
	return compress(targetFile, true, paths)
}

func RootDirNotIncluded(targetFile *os.File, paths ...string) (file *os.File, err error) {
	return compress(targetFile, false, paths)
}

func compress(targetFile *os.File, keepRootDir bool, paths []string) (file *os.File, err error) {
	if len(paths) == 0 {
		return nil, errors.New("[compress] source must be provided")
	}
	for _, path := range paths {
		err = validatePath(path)
		if err != nil {
			return nil, err
		}
	}

	//use existing file
	file = targetFile
	if file == nil {
		file, err = ioutil.TempFile("/tmp", "sealer_compress")
		if err != nil {
			return nil, errors.New("create tmp compress file failed")
		}
	}

	defer func() {
		// TODO this would delete existing file, is that ok?
		if err != nil {
			utils.CleanFile(file)
		}
	}()

	zr := gzip.NewWriter(file)
	tw := tar.NewWriter(zr)
	defer func() {
		_ = tw.Close()
		_ = zr.Close()
	}()

	for _, path := range paths {
		var (
			fi        os.FileInfo
			newFolder string
		)
		if keepRootDir {
			fi, err = os.Stat(path)
			if err != nil {
				return nil, err
			}
			if fi.IsDir() {
				newFolder = filepath.Base(path)
			}
		}

		err = writeToTarWriter(path, newFolder, tw)
		if err != nil {
			return nil, err
		}
	}

	return file, nil
}

func writeToTarWriter(dir, newFolder string, tarWriter *tar.Writer) error {
	dir = strings.TrimSuffix(dir, "/")
	srcPrefix := filepath.ToSlash(dir + "/")
	err := filepath.Walk(dir, func(file string, fi os.FileInfo, err error) error {
		// generate tar header
		header, walkErr := tar.FileInfoHeader(fi, file)
		if walkErr != nil {
			return walkErr
		}
		if file != dir {
			absPath := filepath.ToSlash(file)
			header.Name = filepath.Join(newFolder, strings.TrimPrefix(absPath, srcPrefix))
		} else {
			// do not contain root dir
			if fi.IsDir() {
				return nil
			}
			// for supporting tar single file
			header.Name = filepath.Join(newFolder, filepath.Base(dir))
		}

		// those two fields may diff in different machine,
		// so we hardcoded the gname and uname to make hash of
		// tar consistent
		// TODO think of a better way to fix this issue
		header.Gname = "root"
		header.Uname = "root"
		// write header
		if walkErr = tarWriter.WriteHeader(header); walkErr != nil {
			return walkErr
		}
		// if not a dir, write file content
		if !fi.IsDir() {
			data, walkErr := os.Open(file)
			if walkErr != nil {
				return walkErr
			}
			if _, walkErr = io.Copy(tarWriter, data); walkErr != nil {
				return walkErr
			}
		}
		return nil
	})

	return err
}

// Decompress this will not change the metadata of original files
func Decompress(src io.Reader, dst string) error {
	// need to set umask to be 000 for current process.
	// there will be some files having higher permission like 777,
	// eventually permission will be set to 755 when umask is 022.
	_, err := system.Umask(0)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dst, common.FileMode0755)
	if err != nil {
		return err
	}

	zr, err := gzip.NewReader(src)
	if err != nil {
		return err
	}

	tr := tar.NewReader(zr)
	type DirStruct struct {
		header     *tar.Header
		dir        string
		next, prev *DirStruct
	}

	prefixes := make(map[string]*DirStruct)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// validate name against path traversal
		if !validRelPath(header.Name) {
			return fmt.Errorf("tar contained invalid name error %q", header.Name)
		}

		target := filepath.Join(dst, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err = os.Stat(target); err != nil {
				if err = os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
					return err
				}

				// building a double linked list
				prefix := filepath.Dir(target)
				prev := prefixes[prefix]
				//an root dir
				if prev == nil {
					prefixes[target] = &DirStruct{header: header, dir: target, next: nil, prev: nil}
				} else {
					newHead := &DirStruct{header: header, dir: target, next: nil, prev: prev}
					prev.next = newHead
					prefixes[target] = newHead
				}
			}

		case tar.TypeReg:
			err = func() error {
				// regularly won't mkdir, unless add newFolder on compressing
				inErr := utils.MkDirIfNotExists(filepath.Dir(target))
				if inErr != nil {
					return inErr
				}

				fileToWrite, inErr := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.FileMode(header.Mode))
				if inErr != nil {
					return inErr
				}

				defer fileToWrite.Close()
				if _, inErr = io.Copy(fileToWrite, tr); inErr != nil {
					return inErr
				}
				// for not changing
				return os.Chtimes(target, header.AccessTime, header.ModTime)
			}()

			if err != nil {
				return err
			}
		}
	}

	for _, v := range prefixes {
		// for taking the last one
		if v.next != nil {
			continue
		}

		// every change in dir, will change the metadata of that dir
		// change times from the last one
		// do this is for not changing metadata of parent dir
		for dirStr := v; dirStr != nil; dirStr = dirStr.prev {
			if err = os.Chtimes(dirStr.dir, dirStr.header.AccessTime, dirStr.header.ModTime); err != nil {
				return err
			}
		}
	}

	return nil
}

// check for path traversal and correct forward slashes
func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}
