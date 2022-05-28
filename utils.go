package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	netUrl "net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
)

func readConfig(config_path string) (*RootConfig, error) {
	jsonFile, err := os.Open(config_path)
	fatalLog("Failed to read config.", err)
	defer jsonFile.Close()

	var result RootConfig
	err = json.NewDecoder(jsonFile).Decode(&result)
	if err != nil {
		log.Fatal("Failed to decode config.", err)
		return nil, err
	}

	return &result, nil
}

func downloadUrl(name string, url string) (string, error) {
	l := log.Child("downloadUrl").Child(name)
	l.Debugf("Start download url \"%s\"", url)

	resp, err := http.Get(url)
	if err != nil {
		l.Error(fmt.Sprintf("Failed to download url \"%s\"", url), err)
		return "", err
	}
	defer resp.Body.Close()

	ext := tryDetectExt(url)
	filename := fmt.Sprintf("%s/%s%s", tempDir, name, ext)
	out, err := os.Create(filename)
	if err != nil {
		l.Error(fmt.Sprintf("Failed to create file \"%s\".", filename), err)
		return "", err
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		l.Error(fmt.Sprintf("Failed to write file \"%s\".", filename), err)
		return "", err
	}
	defer out.Close()

	l.Debugf("Finish download url \"%s\"", url)
	return filename, nil
}

func tryDetectExt(url string) string {
	u, err := netUrl.Parse(url)
	if err != nil {
		ext := path.Ext(filepath.Base(url))
		if strings.HasSuffix(url, ".tar"+ext) {
			return ".tar" + ext
		}
		return ext
	}
	ext := path.Ext(u.Path)
	if strings.HasSuffix(url, ".tar"+ext) {
		return ".tar" + ext
	}
	return ext
}

func unArchive(name string, filename string) error {
	l := log.Child("unArchive").Child(name)

	target := fmt.Sprintf("%s/%s", targetDir, name)

	ext := tryDetectExt(filename)
	l.Debugf("Detected extension: %s", ext)
	e := ext[:strings.LastIndex(ext, ".")] // for tar.gz or tar.xz...
	if e == "" {
		e = ext
	}
	switch e {
	case "tar": // .tar
		a := archiver.Tar{
			StripComponents:   1,
			OverwriteExisting: true,
			MkdirAll:          true,
		}
		return a.Unarchive(filename, target)
	case ".tar": // .tar.gz or .tar.xz...
		n := fmt.Sprintf("%s/%s.tar", tempDir, name)
		err := DecompressFileWrapper(filename, n)
		if err != nil {
			return err
		}

		a := archiver.Tar{
			StripComponents:   1,
			OverwriteExisting: true,
			MkdirAll:          true,
		}
		return a.Unarchive(n, target)
	case ".zip":
		a := archiver.Zip{
			StripComponents:   1,
			OverwriteExisting: true,
			MkdirAll:          true,
		}
		return a.Unarchive(filename, target)
	case ".rar":
		a := archiver.Rar{
			StripComponents:   1,
			OverwriteExisting: true,
			MkdirAll:          true,
		}
		return a.Unarchive(filename, target)
	default:
		return archiver.Unarchive(filename, target)
	}
}

func DecompressFileWrapper(source, destination string) error {
	// archiver.DecompressFile has a bug that tar compressed doesn't recognized well... Fix in place.
	cIface, err := archiver.ByExtension(strings.Replace(source, ".tar.", ".", 1))
	if err != nil {
		return err
	}
	c, ok := cIface.(archiver.Decompressor)
	if !ok {
		return fmt.Errorf("format specified by source filename is not a recognized compression algorithm: %s", source)
	}
	return archiver.FileCompressor{Decompressor: c}.DecompressFile(source, destination)
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}
