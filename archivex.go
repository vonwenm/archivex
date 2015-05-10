//////////////////////////////////////////
// archivex.go
// Jhonathan Paulo Banczek - 2014
// jpbanczek@gmail.com - jhoonb.com
//////////////////////////////////////////

package archivex

import (
	"archive/tar"
	"archive/zip"
	"io"
	"io/ioutil"
	// "log"
	"os"
	"path"
	"strings"
)

// interface
type Archivex interface {
	Create(name string) error
	Add(name string, file []byte) error
	AddFile(name string) error
	AddAll(dir string, includeCurrentFolder bool) error
	Close() error
}

type ArchiveWriteFunc func(info os.FileInfo, file io.Reader, entryName string) (err error)

// ZipFile implement *zip.Writer
type ZipFile struct {
	Writer *zip.Writer
	Name   string
}

// TarFile implement *tar.Writer
type TarFile struct {
	Writer *tar.Writer
	Name   string
}

// Create new file zip
func (z *ZipFile) Create(name string) error {
	// check extension .zip
	if strings.HasSuffix(name, ".zip") != true {
		if strings.HasSuffix(name, ".tar.gz") == true {
			name = strings.Replace(name, ".tar.gz", ".zip", -1)
		} else {
			name = name + ".zip"
		}
	}
	z.Name = name
	file, err := os.Create(z.Name)
	if err != nil {
		return err
	}
	z.Writer = zip.NewWriter(file)
	return nil
}

// Add add byte in archive zip
func (z *ZipFile) Add(name string, file []byte) error {

	iow, err := z.Writer.Create(name)
	if err != nil {
		return err
	}
	_, err = iow.Write(file)
	return err
}

// AddFile add file from dir in archive
func (z *ZipFile) AddFile(name string) error {
	bytearq, err := ioutil.ReadFile(name)
	if err != nil {
		return err
	}
	filep, err := z.Writer.Create(name)
	if err != nil {
		return err
	}
	_, err = filep.Write(bytearq)
	if err != nil {
		return err
	}
	return nil
}

// AddAll add all files from dir in archive
func (z *ZipFile) AddAll(dir string, includeCurrentFolder bool) error {
	dir = path.Clean(dir)
	return addAll(dir, dir, includeCurrentFolder, func(info os.FileInfo, file io.Reader, entryName string) (err error) {

		// Create a header based off of the fileinfo
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Set the header's name to what we want--it may not include the top folder
		header.Name = entryName

		// Get a writer in the archive based on our header
		writer, err := z.Writer.CreateHeader(header)
		if err != nil {
			return err
		}

		// Pipe the file into the archive writer
		if _, err := io.Copy(writer, file); err != nil {
			return err
		}

		return nil
	})
}

func (z *ZipFile) Close() error {
	err := z.Writer.Close()
	return err
}

// Create new Tar file
func (t *TarFile) Create(name string) error {
	// check extension .zip
	if strings.HasSuffix(name, ".tar.gz") != true {
		if strings.HasSuffix(name, ".zip") == true {
			name = strings.Replace(name, ".zip", ".tar.gz", -1)
		} else {
			name = name + ".tar.gz"
		}
	}
	t.Name = name
	file, err := os.Create(t.Name)
	if err != nil {
		return err
	}
	t.Writer = tar.NewWriter(file)
	return nil
}

// Add add byte in archive tar
func (t *TarFile) Add(name string, file []byte) error {

	hdr := &tar.Header{Name: name, Size: int64(len(file))}
	if err := t.Writer.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := t.Writer.Write(file)
	return err
}

// AddFile add file from dir in archive tar
func (t *TarFile) AddFile(name string) error {
	bytearq, err := ioutil.ReadFile(name)
	if err != nil {
		return err
	}

	hdr := &tar.Header{Name: name, Size: int64(len(bytearq))}
	err = t.Writer.WriteHeader(hdr)
	if err != nil {
		return err
	}
	_, err = t.Writer.Write(bytearq)
	if err != nil {
		return err
	}
	return nil

}

// AddAll add all files from dir in archive
func (t *TarFile) AddAll(dir string, includeCurrentFolder bool) error {
	dir = path.Clean(dir)
	return addAll(dir, dir, includeCurrentFolder, func(info os.FileInfo, file io.Reader, entryName string) (err error) {

		// Create a header based off of the fileinfo
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Set the header's name to what we want--it may not include the top folder
		header.Name = entryName

		// Write the header into the tar file
		if err := t.Writer.WriteHeader(header); err != nil {
			return err
		}

		// Pipe the file into the tar
		if _, err := io.Copy(t.Writer, file); err != nil {
			return err
		}

		return nil
	})
}

// Close the file Tar
func (t *TarFile) Close() error {
	err := t.Writer.Close()
	return err
}

func getSubDir(dir string, rootDir string, includeCurrentFolder bool) (subDir string) {

	subDir = strings.Replace(dir, rootDir, "", 1)

	if includeCurrentFolder {
		parts := strings.Split(rootDir, string(os.PathSeparator))
		subDir = path.Join(parts[len(parts)-1], subDir)
	}

	return
}

// addAll is used to recursively go down through directories and add each file to an archive, based on an ArchiveWriteFunc given to it
func addAll(dir string, rootDir string, includeCurrentFolder bool, writerFunc ArchiveWriteFunc) error {

	// Get a list of all entries in the directory, as []os.FileInfo
	listFile, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	// Prepare a list of all non-directory files in the current directory
	nonDirs := []os.FileInfo{}

	// Loop through all files in the current directory
	for _, arq := range listFile {
		full := path.Join(dir, arq.Name())
		if arq.IsDir() {

			// For each directory, recurse into it
			addAll(full, rootDir, includeCurrentFolder, writerFunc)

		} else {

			// Otherwise, add the file to the list of things to write into the archive
			nonDirs = append(nonDirs, arq)

		}
	}

	// Now we loop through all of our non-directory files and write them into the archive
	subDir := getSubDir(dir, rootDir, includeCurrentFolder)
	for _, nonDir := range nonDirs {

		// Open the file we're going to write into the archive
		full := path.Join(dir, nonDir.Name())
		file, err := os.Open(full)
		if err != nil {
			return err
		}
		defer func() {
			file.Close()
		}()

		entryName := path.Join(subDir, nonDir.Name())
		if err := writerFunc(nonDir, file, entryName); err != nil {
			return err
		}
	}

	return nil
}
