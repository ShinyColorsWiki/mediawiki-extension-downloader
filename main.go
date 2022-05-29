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

var log = golog.New()

var MWREL string = "master" // this does for default branch name too.
var targetDir string
var tempDir string
var hasError = false // check has error during download

func fatalLog(msg string, err error) {
	if err != nil {
		hasError = true
		log.Fatal(msg, err)
	}
}

func main() {
	// Set log level from environment
	log.SetLevel(getEnv("LOG_LEVEL", "info"))

	// Defined and parse flags
	config_path := flag.String("config", "./config.json", "A config file for download extensions and skins.")
	target_path := flag.String("target", "./downloaded", "A target folder for downloaded extensions and skins.")
	force_rm_target := flag.Bool("force-rm-target", false, "Turn this on to delete target directory if exist. Be careful to use!")
	flag.Parse()

	// Set flags to each variables.
	config, err := readConfig(*config_path)
	fatalLog("Error during read config.", err)

	targetDir, err = filepath.Abs(*target_path)
	fatalLog("Failed to convert target path to absoluete path.", err)

	// Set MWREL for download.
	MWREL = getEnv("MWREL", config.MWREL)
	if MWREL == "" {
		MWREL = "master" // fallback to master.
	}

	// Make Target Directories.
	mkTargetDir := func() {
		err = os.MkdirAll(targetDir, os.ModePerm)
		fatalLog("Failed to create target path directory", err)
	}

	mkExtSkinDir := func() {
		err = os.MkdirAll(targetDir+"/extensions", os.ModePerm)
		fatalLog("Failed to create extensions directory inside of target path directory", err)
		err = os.MkdirAll(targetDir+"/skins", os.ModePerm)
		fatalLog("Failed to create skins directory inside of target path directory", err)
	}

	// Check Target Directory exists. if not, make directory.
	if isDirNotExist(targetDir) {
		mkTargetDir()
	}

	// Check extensions and skins directory exists. if not make one, if it is check flag for deletion
	if isDirNotExist(targetDir+"/extensions") || isDirNotExist(targetDir+"/skins") {
		mkExtSkinDir()
	} else if !*force_rm_target {
		msg := "Target folder already exist. Please remove your self or set '--force-rm-target=true'"
		fatalLog(msg, errors.New(msg))
	} else {
		err = os.RemoveAll(targetDir + "/extensions")
		fatalLog("Failed to clean up target/extensions path directory", err)
		err = os.RemoveAll(targetDir + "/skins")
		fatalLog("Failed to clean up target/skins path directory", err)
		mkExtSkinDir()
	}

	// Make temporal directory for download and extract.
	tempDir, err = os.MkdirTemp("", "mediawiki-extension-downloader-")
	fatalLog("Failed to generate temporal directory.", err)

	log.Info("Starting downloader...")

	// Downloader.
	var wg sync.WaitGroup
	DownloadTargets := parseConfigToUrls(*config)
	wg.Add(len(DownloadTargets))
	for _, opts := range DownloadTargets {
		log.Debugf("Start download %s \"%s\"", opts.Type, opts.Name)
		go opts.StartDownload(&wg)
	}
	wg.Wait()

	log.Debug("Cleanup temp directory...")
	os.RemoveAll(tempDir)
	fatalLog("Failed to delete temporal folder", err)

	// Finish notify
	if hasError {
		log.Info("Error has occured during download. Please check the logs.")
	}
	log.Info("Download Finished.")
}

func (o DownloadOption) StartDownload(wg *sync.WaitGroup) {
	defer wg.Done()

	// remove "s" suffix
	target_name := o.Type[:len(o.Type)-1]

	// Download file from url.
	filename, err := downloadUrl(o.Name, o.Url)
	if err != nil {
		msg := fmt.Sprintf("Failed to download %s \"%s\" ", target_name, o.Name)
		log.Error(msg, err)
		hasError = true
	}

	// Extract to temp directory.
	dirpath, err := unArchive(o.Name, filename)
	if err != nil {
		msg := fmt.Sprintf("Failed to extract %s \"%s\" to \"%s\" ", target_name, o.Name, dirpath)
		log.Error(msg, err)
		hasError = true
	}

	// And move to target.
	dest := fmt.Sprintf("%s/%s/%s", targetDir, o.Type, o.Name)
	err = os.Rename(dirpath, dest)
	if err != nil {
		msg := fmt.Sprintf("Failed to move %s \"%s\" from \"%s\" to \"%s\"", target_name, o.Name, dirpath, dest)
		log.Error(msg, err)
		hasError = true
	}
}
