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
	MWREL      string     `json:"MWREL,omitempty"`
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
	return DownloadOption{Type: t, Name: name, Url: strings.ReplaceAll(url, "$mwrel", MWREL)}
}

// Parse config to make download options.
func parseConfigToUrls(config RootConfig) []DownloadOption {
	var result []DownloadOption
	a := func(t DownloadOption) {
		result = append(result, t)
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
		a(NewDownloadOptions("extensions", name, url))
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
		a(NewDownloadOptions("skins", name, url))
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
	return requestWMFExtDistUrl("extension", name)
}

// Make WMF Skin Download Url.
func WMFSkinUrl(name string) string {
	return requestWMFExtDistUrl("skin", name)
}

type GerritBranchResponse struct {
	// Much more but only needed.
	Revision string `json:"revision"`
}

// TODO: async-able.
func requestWMFExtDistUrl(t string, name string) string {
	errHandler := func(msg string, err error) string {
		log.Child("requestWMFExtDistUrl").Child(name).Error(msg, err)
		hasError = true
		return fmt.Sprintf("https://github.com/wikimedia/mediawiki-%ss-%s/archive/%s.tar.gz", t, name, MWREL)
	}

	// FIXME: Use gitlab when they moved.
	branchInfoUrl := fmt.Sprintf("https://gerrit.wikimedia.org/r/projects/mediawiki%%2F%ss%%2F%s/branches/%s", t, name, MWREL)
	resp, err := http.Get(branchInfoUrl)
	if err != nil {
		return errHandler("WMF Extension distributor infomation request failed. ", err)
	}

	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errHandler("WMF Extension distributor infomation response read failed. ", err)
	}

	bytes = bytes[5 : len(bytes)-1]
	var branchInfo GerritBranchResponse
	err = json.Unmarshal(bytes, &branchInfo)
	if err != nil {
		return errHandler("WMF Extension distributor infomation response decode failed. ", err)
	}

	return fmt.Sprintf("https://extdist.wmflabs.org/dist/%ss/%s-%s-%s.tar.gz", t, name, MWREL, branchInfo.Revision[0:9])
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

// Check is Directory not exist.
func isDirNotExist(p string) bool {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return true
	}
	return false
}
