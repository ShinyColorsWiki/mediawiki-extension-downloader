package main

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"os"
	"strings"

	"github.com/nwaples/rardecode"
)

// Check if the file is a valid archive file (zip, tar.gz, tar.xz)
func isValidArchiveFile(filename string) bool {
	l := log.Child("isValidArchiveFile")

	if strings.HasSuffix(filename, ".zip") {
		l.Debugf("Validating ZIP file: %s", filename)
		return isValidZip(filename)
	} else if strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz") {
		l.Debugf("Validating TAR.GZ file: %s", filename)
		return isValidTarGz(filename)
	} else if strings.HasSuffix(filename, ".tar.bz2") {
		l.Debugf("Validating TAR.BZ2 file: %s", filename)
		return isValidTarBz2(filename)
	} else if strings.HasSuffix(filename, ".tar") {
		l.Debugf("Validating TAR file: %s", filename)
		return isValidTar(filename)
	} else if strings.HasSuffix(filename, ".rar") {
		l.Debugf("Validating RAR file: %s", filename)
		return isValidRar(filename)
	}
	// Add more formats as needed
	return false
}

// Validate ZIP files
func isValidZip(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return false
	}

	_, err = zip.NewReader(file, stat.Size())
	return err == nil
}

// Validate .tar.gz files
func isValidTarGz(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return false
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	_, err = tarReader.Next() // Try reading first file entry
	return err == nil
}

// Validate .tar.bz2 files
func isValidTarBz2(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	bzip2Reader := bzip2.NewReader(file)
	tarReader := tar.NewReader(bzip2Reader)
	_, err = tarReader.Next() // Try reading first file entry
	return err == nil
}

// Validate .tar files
func isValidTar(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	tarReader := tar.NewReader(file)
	_, err = tarReader.Next() // Try reading first file entry
	return err == nil
}

// Validate .rar files
func isValidRar(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	rarReader, err := rardecode.NewReader(file, "")
	if err != nil {
		return false
	}

	_, err = rarReader.Next() // Try reading first file entry
	return err == nil
}
