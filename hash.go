package zigcentral

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/unix"
)

type HashedFile struct {
	Path string
	Hash []byte
}

type HashedFiles []HashedFile

func (a HashedFiles) Len() int           { return len(a) }
func (a HashedFiles) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a HashedFiles) Less(i, j int) bool { return strings.Compare(a[i].Path, a[j].Path) == -1 }

func ComputeHash(directory string) string {
	if directory[len(directory)-1] != '/' {
		directory = directory + "/"
	}

	hashedFiles := make([]HashedFile, 0)
	filepath.Walk(directory, func(path string, info fs.FileInfo, err error) error {
		relativePath := strings.TrimPrefix(path, directory)
		if info.IsDir() {
			return err
		}

		var executable byte = 0
		if unix.Access(path, unix.X_OK) == nil {
			executable = 1
		}

		fileHasher := sha256.New()
		fileHasher.Write([]byte(relativePath))
		fileHasher.Write([]byte{0, executable})

		file, ferr := os.Open(path)
		if ferr != nil {
			return ferr
		}
		defer file.Close()

		io.Copy(fileHasher, file)

		hashedFiles = append(hashedFiles, HashedFile{Path: path, Hash: fileHasher.Sum(nil)})

		return err
	})

	sort.Sort(HashedFiles(hashedFiles))

	globalHasher := sha256.New()
	for _, hf := range hashedFiles {
		globalHasher.Write(hf.Hash)
	}
	digest := globalHasher.Sum(nil)

	// sha256 - is 12, digest length - 20
	return "1220" + hex.EncodeToString(digest[:])
}
