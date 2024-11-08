package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

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
	retry_count := flag.Int("retry-count", 3, "Number of retries for download and extraction process.")
	retry_delay := flag.Int("retry-delay", 2, "Delay in seconds between retries for download and extraction process.")
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
		go opts.StartDownload(&wg, *retry_count, time.Duration(*retry_delay)*time.Second)
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

func (o DownloadOption) StartDownload(wg *sync.WaitGroup, retryCount int, retryDelay time.Duration) {
	defer wg.Done()

	err := retry(func() error {
		// remove "s" suffix
		log.Info(o.Type)
		targetName := o.Type[:len(o.Type)-1]

		// Step 1: Download file from URL
		filename, err := downloadUrl(o.Name, o.Url)
		if err != nil {
			return fmt.Errorf("Failed to download %s \"%s\": %w", targetName, o.Name, err)
		}

		// Step 2: Check if the file is a valid archive
		if !isValidArchiveFile(filename) {
			return fmt.Errorf("Downloaded file is not an archive: %s \"%s\"", targetName, o.Name)
		}

		// Step 3: Extract the file to a temporary directory
		dirpath, err := unArchive(o.Name, filename)
		if err != nil {
			return fmt.Errorf("Failed to extract %s \"%s\": %w", targetName, o.Name, err)
		}

		// Step 4: Move the extracted directory to the target location
		dest := fmt.Sprintf("%s/%s/%s", targetDir, o.Type, o.Name)
		err = os.Rename(dirpath, dest)
		if err != nil {
			return fmt.Errorf("Failed to move %s \"%s\" to \"%s\": %w", targetName, o.Name, dest, err)
		}

		return nil
	}, retryCount, retryDelay, fmt.Sprintf("Complete download and extraction process for \"%s\"", o.Name))

	if err != nil {
		log.Error(err)
		hasError = true
	}
}

// retry function to handle retries with delay for entire process
func retry(operation func() error, attempts int, delay time.Duration, description string) error {
	for i := 0; i < attempts; i++ {
		err := operation()
		if err == nil {
			return nil
		}
		if i < attempts-1 {
			log.Warnf("Retrying %s... attempt %d", description, i+2)
			time.Sleep(delay)
		} else {
			return fmt.Errorf("failed to %s after %d attempts: %w", description, attempts, err)
		}
	}
	return nil
}
