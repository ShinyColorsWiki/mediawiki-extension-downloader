package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	netUrl "net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
)

// Root of config
type RootConfig struct {
	MWREL      string     `json:"MWREL"`
	Extensions TypeConfig `json:"Extensions"`
	Skins      TypeConfig `json:"Skins"`
}

// The download types
type TypeConfig struct {
	WMF  []string             `json:"WMF,omitempty"`
	Git  map[string]GitConfig `json:"Git,omitempty"`
	Http map[string]string    `json:"http,omitempty"`
}

// Git Type Config
type GitConfig struct {
	Type    string `json:"type"`
	Repo    string `json:"repo,omitempty"`
	Branch  string `json:"branch,omitempty"`
	RepoUrl string `json:"repoUrl,omitempty"`
}

// Parsed download option.
type DownloadOption struct {
	Type string
	Name string
	Url  string
}

// Parse config to struct.
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

// Just make new DownloadOption
func NewDownloadOptions(t string, name string, url string) DownloadOption {
	return DownloadOption{Type: t, Name: name, Url: url}
}

// Parse config to make download options.
func parseConfigToUrls(config RootConfig) []DownloadOption {
	var result []DownloadOption
	a := func(t DownloadOption) {
		result = append(result, t)
	}

	u := func(url string) string {
		return strings.ReplaceAll(url, "$mwrel", MWREL)
	}

	for _, name := range config.Extensions.WMF {
		a(NewDownloadOptions("extensions", name, WMFExtensionUrl(name)))
		log.Debugf("Added WMF Extension \"%s\" to download queue.", name)
	}

	for name, git := range config.Extensions.Git {
		a(NewDownloadOptions("extensions", name, git.MakeGitUrl()))
		log.Debugf("Added Git Extension \"%s\" to download queue.", name)
	}

	for name, url := range config.Extensions.Http {
		a(NewDownloadOptions("extensions", name, u(url)))
		log.Debugf("Added Url Extension \"%s\" to download queue.", name)
	}

	for _, name := range config.Skins.WMF {
		a(NewDownloadOptions("skins", name, WMFSkinUrl(name)))
		log.Debugf("Added WMF Skin \"%s\" to download queue.", name)
	}

	for name, git := range config.Skins.Git {
		a(NewDownloadOptions("skins", name, git.MakeGitUrl()))
		log.Debugf("Added Git Skin \"%s\" to download queue.", name)
	}

	for name, url := range config.Skins.Http {
		a(NewDownloadOptions("skins", name, u(url)))
		log.Debugf("Added Url Skin \"%s\" to download queue.", name)
	}

	return result
}

// Make Git Url from GitConfig
func (s GitConfig) MakeGitUrl() string {
	// TODO: Direct Git Download for not-defacto git sites like self-hosted git solutions.
	// No direct git download when using url builder.

	// Choose default branch
	if s.Branch == "" {
		s.Branch = MWREL
	}

	switch s.Type {
	case "github":
		return fmt.Sprintf("https://github.com/%s/archive/%s.tar.gz", s.Repo, s.Branch)
	case "gitlab":
		return fmt.Sprintf("https://gitlab.com/%s/-/archive/%s.tar.gz", s.Repo, s.Branch) // TODO: Is this really works well?
	default:
		return ""
	}
}

// Make WMF Extension Download Url.
func WMFExtensionUrl(name string) string {
	// FIXME: Use gerrit. or gitlab when they moved.
	return fmt.Sprintf("https://github.com/wikimedia/mediawiki-extensions-%s/archive/%s.tar.gz", name, MWREL)
}

// Make WMF Skin Download Url.
func WMFSkinUrl(name string) string {
	// FIXME: Use gerrit. or gitlab when they moved.
	return fmt.Sprintf("https://github.com/wikimedia/mediawiki-skins-%s/archive/%s.tar.gz", name, MWREL)
}

// Handle Download
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

// Try to detect extension for extract.
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

// UnArchive using archiver.
func unArchive(name string, filename string) (string, error) {
	l := log.Child("unArchive").Child(name)

	target := fmt.Sprintf("%s/%s-%d", tempDir, name, rand.Int())

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
		return target, a.Unarchive(filename, target)
	case ".tar": // .tar.gz or .tar.xz...
		n := fmt.Sprintf("%s/%s.tar", tempDir, name)
		err := DecompressFileWrapper(filename, n)
		if err != nil {
			return "", err
		}

		a := archiver.Tar{
			StripComponents:   1,
			OverwriteExisting: true,
			MkdirAll:          true,
		}
		return target, a.Unarchive(n, target)
	case ".zip":
		a := archiver.Zip{
			StripComponents:   1,
			OverwriteExisting: true,
			MkdirAll:          true,
		}
		return target, a.Unarchive(filename, target)
	case ".rar":
		a := archiver.Rar{
			StripComponents:   1,
			OverwriteExisting: true,
			MkdirAll:          true,
		}
		return target, a.Unarchive(filename, target)
	default:
		return target, archiver.Unarchive(filename, target)
	}
}

// archiver.DecompressFile has a bug that tar compressed doesn't recognized well... Fix in place.
func DecompressFileWrapper(source, destination string) error {
	// this line is fix.
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

// Get Environment value with fallback
func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}
