package zigcentral

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
)

func ExtractTarGz(gzipStream io.Reader) (directory string, err error) {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return "", err
	}
	rootDir := ""

	directory, err = os.MkdirTemp("", "pkgs")
	if err != nil {
		return "", err
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return "", err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if rootDir == "" {
				rootDir = header.Name
			}
			if err := os.Mkdir(directory+"/"+header.Name, 0755); err != nil {
				return "", err
			}
		case tar.TypeReg:
			outFile, err := os.Create(directory + "/" + header.Name)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return "", err
			}
			outFile.Close()
		default:
			continue
		}
	}

	return directory + "/" + rootDir, nil
}
