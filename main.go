package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/kataras/golog"
)

type RootConfig struct {
	MWREL string                  `json:"MWREL"`
	WMF   []string                `json:"WMF,omitempty"`
	Git   map[string]GitExtConfig `json:"Git,omitempty"`
	Http  map[string]string       `json:"http,omitempty"`
}

type GitExtConfig struct {
	Type    string `json:"type"`
	Repo    string `json:"repo,omitempty"`
	Branch  string `json:"branch,omitempty"`
	RepoUrl string `json:"repoUrl,omitempty"`
}

var log = golog.New()

var MWREL string = "master" // this does for default branch name too.
var targetDir string
var tempDir string
var hasError = false

func fatalLog(msg string, err error) {
	if err != nil {
		hasError = true
		log.Fatal(msg, err)
	}
}

func main() {
	log.SetLevel(getEnv("LOG_LEVEL", "info"))

	config_path := flag.String("config", "./config.json", "A config file for download extensions.")
	target_path := flag.String("target", "./dowloaded-extension", "A target folder for downloaded extensions.")
	force_rm_target := flag.Bool("force-rm-target", false, "Turn this on to delete target directory if exist. Be careful to use!")
	flag.Parse()

	config, err := readConfig(*config_path)
	fatalLog("Error during read config.", err)

	targetDir, err = filepath.Abs(*target_path)
	fatalLog("Failed to convert target path to absoluete path.", err)

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		err = os.MkdirAll(targetDir, os.ModePerm)
		fatalLog("Failed to create target path directory", err)
	} else if !*force_rm_target {
		msg := "Target folder already exist. Please remove your self or set '--force-rm-target=true'"
		fatalLog(msg, errors.New(msg))
	} else {
		err := os.RemoveAll(targetDir)
		fatalLog("Failed to create target path directory", err)
		err = os.MkdirAll(targetDir, os.ModePerm)
		fatalLog("Failed to create target path directory", err)
	}

	tempDir, err = os.MkdirTemp("", "mediawiki-extension-downloader-")
	fatalLog("Failed to generate temporal directory.", err)

	log.Info("Starting downloader...")

	var wg sync.WaitGroup
	wg.Add(len(config.WMF))
	for _, wmf_ext := range config.WMF {
		log.Debugf("Starting WMF Extension Downloader for \"%s\"", wmf_ext)
		go WMFDownloader(wmf_ext, &wg)
	}

	wg.Add(len(config.Git))
	for name, git := range config.Git {
		log.Debugf("Starting Git Downloader for \"%s\"", name)
		go git.GitExtDownloader(name, &wg)
	}

	wg.Add(len(config.Http))
	for name, url := range config.Http {
		log.Debugf("Starting Url Downloader for \"%s\" (\"%s\")", name, url)
		go HttpDownloader(name, url, &wg)
	}

	wg.Wait()

	log.Debug("Cleanup temp directory...")

	os.RemoveAll(tempDir)
	fatalLog("Failed to delete temporal folder", err)

	log.Info("Download Finished.")
}

func (s GitExtConfig) GitExtDownloader(name string, wg *sync.WaitGroup) {
	// TODO: Direct Git Download for not-defacto git sites like self-hosted git solutions.
	// No direct git download when using url builder.
	buildUrl := func() string {
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

	HttpDownloader(name, buildUrl(), wg)
}

func WMFDownloader(name string, wg *sync.WaitGroup) {
	// FIXME: Use gerrit. or gitlab when they moved.
	url := fmt.Sprintf("https://github.com/wikimedia/mediawiki-extensions-%s/archive/%s.tar.gz", name, MWREL)
	HttpDownloader(name, url, wg)
}

func HttpDownloader(name string, url string, wg *sync.WaitGroup) {
	defer wg.Done()

	filename, err := downloadUrl(name, url)
	if err != nil {
		msg := fmt.Sprintf("Failed to download extension \"%s\" ", name)
		log.Error(msg, err)
		hasError = true
	}

	err = unArchive(name, filename)
	if err != nil {
		msg := fmt.Sprintf("Failed to extract extension \"%s\" ", name)
		log.Error(msg, err)
		hasError = true
	}
}
