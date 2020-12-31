/*
Package imaget provides a convenient image tool for finding images on any http(s) website and
downloading them with optional parameters to tweak behaviour and output.
*/
package imaget

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/cavaliercoder/grab"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Client is the http client to use.
var Client = http.DefaultClient

// Std io
var (
	Stdout io.Writer = os.Stdout
	Stderr io.Writer = os.Stderr
)

// Download holds all parameters to Start a download.
type Download struct {
	Src string // The source URL to find and download images from.
	Dst string // The destination to place the downloaded images at.

	Regex      *regexp.Regexp // A regex to filter images.
	SkipAccept bool           // Whether to skip the accept screen before downloading.
	// Whether to save the images flat, instead of creating
	// subdirectories as per the image download URLs.
	SaveFlat bool
	Bar      ProgressBar
}

// ProgressBar can show a progress bar for a download.
type ProgressBar interface {
	Start()           // Start showing the bar.
	Finish()          // Finish and hide the bar.
	SetTotal(int64)   // Set the maximum value.
	SetCurrent(int64) // Set the current value.
}

// Start searches for images on the specified website Src and
// downloads matching images to the desired destination Dst.
// Canceling the context will only pause the download and can
// be resumed to proceed downloading where paused at.
func (d *Download) Start(ctx context.Context) error {
	// Prepare images destination
	dst, err := newDst(d.Dst)
	if err != nil {
		return err
	}
	defer dst.Close()
	// Create http request
	req, err := newRequest(ctx, d.Src)
	if err != nil {
		return err
	}
	// Read website content
	content, err := d.readSite(req)
	if err != nil {
		return fmt.Errorf("error reading website content: %w", err)
	}
	// Extract matching image urls
	imageURLs := d.extractImageURLs(content)
	fmt.Fprintln(Stdout, "Found", len(imageURLs), "matching", pluralize("image", len(imageURLs)), "on", d.Src)
	fmt.Fprintln(Stdout)
	// Accept screen
	if !d.SkipAccept && !acceptScreen(fmt.Sprintf("Do you want to start downloading to destination %q?", dst)) {
		// Download not accepted
		return nil
	}
	// Download images
	startTime := time.Now()
	defer func() {
		fmt.Fprintln(Stdout, "\nSaved", len(imageURLs),
			pluralize("image", len(imageURLs)),
			"within", time.Since(startTime), "at", dst)
	}()
	files := make(chan file, 3)
	go func() {
		d.downloadImages(ctx, imageURLs, files)
		close(files)
	}()
	// Copy cached downloads to desired destination
	copyFilesToDst(ctx, d.SaveFlat, dst, files)
	return nil
}

// creates http GET request
func newRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}
	req.Header.Set("User-Agent", "Imaget/alpha image downloader")
	return req, nil
}

// sends http request and returns response body
func (d *Download) readSite(req *http.Request) ([]byte, error) {
	// Send http request
	res, err := Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error doing http request: %w", err)
	}
	defer res.Body.Close()
	// Read response body
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading http response body: %w", err)
	}
	return body, nil
}

// regex for http(s) image urls
var imageRegex = regexp.MustCompile(`(http(s?):)([/|.|\w|\s|-])*\.(?:jpg|gif|png)`)

// finds matching image urls
func (d *Download) extractImageURLs(s []byte) []string {
	// Filter all image urls from body
	a := imageRegex.FindAllString(string(s), -1)
	b := make([]string, 0, len(a)) // scratch space
	// deduplicate urls
	c := make(map[string]struct{}, len(a))
	for _, s := range a {
		if _, exists := c[s]; exists {
			continue
		}
		c[s] = struct{}{}
		b = append(b, s)
	}
	c = nil
	a = a[:0] // reset slice, reuse allocated capacity
	// Filter by or regex
	if d.Regex != nil {
		for _, s := range b {
			if d.Regex.MatchString(s) {
				a = append(a, s)
			}
		}
		return a
	}
	return b
}

// console interaction to accept start of images download
func acceptScreen(titel string) (accepted bool) {
	scan := bufio.NewScanner(os.Stdin)
	for {
		fmt.Fprintf(Stdout, "%s (Press y/n): ", titel)
		if !scan.Scan() {
			break
		}
		switch scan.Text() {
		case "y", "yes", "j", "":
			// Accepted
			return true
		case "n", "no":
			return false
		}

	}
	return false
}

// download images from urls to a temporary directory
func (d *Download) downloadImages(ctx context.Context, imageURLs []string, files chan<- file) {
	if len(imageURLs) == 0 {
		return
	}
	// Ticker to update bar progress
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	// Download all images
	for i, url := range imageURLs {
		fmt.Fprintf(Stdout, "(%d/%d) %s\n", i+1, len(imageURLs), url)
		d.Bar.Start()
		// Download image to temporary file
		startTime := time.Now()
		f, err := downloadImage(ctx, url, grab.DefaultClient, d.Bar, t)
		d.Bar.Finish()
		if err != nil {
			fmt.Fprintf(Stderr, "error downloading image: %v\n", err)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			continue
		}
		fmt.Fprintf(Stdout, " Download finished within %s\n", time.Since(startTime))
		files <- file{path: f, url: url}
	}
}

// tmp directory of imaget
var tmpDir = filepath.Join(os.TempDir(), "imaget")

// downloads an image / resumes download from where was stopped last time
// and returns the name of the downloaded file
func downloadImage(ctx context.Context, imageURL string, c *grab.Client, bar ProgressBar, t *time.Ticker) (file string, err error) {
	// Create request
	req, err := grab.NewRequest(filepath.Join(tmpDir, base64Filename(imageURL)), imageURL)
	if err != nil {
		return "", fmt.Errorf("error creating new download request for %q: %w", imageURL, err)
	}
	req = req.WithContext(ctx)
	// Start download
	res := c.Do(req)
	// Download progress
	bar.SetTotal(res.Size)
	defer bar.SetTotal(res.Size)
	bar.SetCurrent(res.BytesComplete())
loop:
	for {
		select {
		case <-t.C:
			bar.SetCurrent(res.BytesComplete())
		case <-res.Done:
			break loop
		}
	}
	if res.Err() != nil {
		return "", fmt.Errorf("error download %q: %w", imageURL, res.Err())
	}
	return res.Filename, nil
}

// copies received files to the destination
func copyFilesToDst(ctx context.Context, flat bool, dst destination, files <-chan file) {
	for {
		select {
		case <-ctx.Done():
			return
		case f, ok := <-files:
			if !ok {
				return
			}
			if err := copyFileToDst(flat, dst, f); err != nil {
				fmt.Fprintf(Stderr, "error copying image to destination: %v\n", err)
			}
		}
	}
}

// copies one file to a destination
func copyFileToDst(flat bool, dst destination, file file) error {
	// Open source file to be copies to destination
	src, err := os.Open(file.path)
	if err != nil {
		return fmt.Errorf("error opening file %s: %w", file.path, err)
	}
	defer src.Close()
	// Path where to copy the file to
	var dstFile string
	if flat {
		dstFile = filepath.Base(file.path)
	} else {
		dstFile = strings.TrimPrefix(file.url, "http://")
		dstFile = strings.TrimPrefix(dstFile, "https://")
	}
	// Create/open file in destination
	f, err := dst.create(dstFile)
	if err != nil {
		return fmt.Errorf("error creating destination file (%s): %w", dstFile, err)
	}
	defer f.Close()
	// Copy file to destination
	_, err = io.Copy(f, src)
	if err != nil {
		return fmt.Errorf("error copying %s to destination %s: %w", dstFile, dst, err)
	}
	return nil
}

// file is a downloaded file with the absolute
// path and the url it has been downloaded from
type file struct {
	path string
	url  string
}

// encodes an image url to base64 to become a valid file name
func base64Filename(imageURL string) string {
	return base64.URLEncoding.EncodeToString([]byte(imageURL)) + filepath.Ext(imageURL)
}

// util to append an 's' to a string if count is 1, 0 or -1
func pluralize(s string, count int) string {
	if count > 1 || count == 0 || count < -1 {
		return s + "s"
	}
	return s
}

// creates a destination to be used to save files into
func newDst(dst string) (destination, error) {
	dst, err := filepath.Abs(dst)
	if err != nil {
		return nil, fmt.Errorf("error getting absolute path of destination: %w", err)
	}
	switch filepath.Ext(dst) {
	case "":
		// Destination will be a directory
		return dirDst(dst), nil
	case ".zip":
		// Destination will be an archive
		// Create folder path upon directory of archive
		if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
			return nil, fmt.Errorf("error creating directory path for archive: %w", err)
		}
		return newZipDst(dst)
	}
	return nil, errors.New("unsupported destination")
}

// destination is a file storage. Call Close when finished.
type destination interface {
	// Creates a new file in the destination to write to.
	// Must be closed after done writing.
	create(file string) (io.WriteCloser, error)
	// Must be called after use of the destination.
	io.Closer
	// The string representation of the destination.
	fmt.Stringer
}

// dirDst is a directory destination
type dirDst string

func (d dirDst) String() string { return string(d) }
func (d dirDst) create(file string) (io.WriteCloser, error) {
	// Create folder path upon file
	file = filepath.Join(string(d), file)
	dirPath := filepath.Dir(file)
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("error creating directory %q: %w", dirPath, err)
	}
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", file, err)
	}
	return f, nil
}
func (d dirDst) Close() error { return nil }

// zipDst is a zip archive destination
type zipDst struct {
	dst string
	f   *os.File
	w   *zip.Writer
}

func newZipDst(dst string) (destination, error) {
	f, err := os.Create(dst)
	if err != nil {
		return nil, fmt.Errorf("error creating destination archive: %w", err)
	}
	return &zipDst{
		dst: dst,
		f:   f,
		w:   zip.NewWriter(f),
	}, nil
}
func (d *zipDst) String() string { return d.dst }
func (d *zipDst) create(file string) (io.WriteCloser, error) {
	f, err := d.w.Create(file)
	return &nopCloser{f}, err
}
func (d *zipDst) Close() error {
	defer d.f.Close()
	return d.w.Close()
}

// nopCloser is an io.Writer implementing a no-operation io.Closer
type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }
