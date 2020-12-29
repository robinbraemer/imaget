package imaget

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/cavaliercoder/grab"
	"github.com/schollz/progressbar/v3"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var Client = http.DefaultClient

// Download holds all parameters to Start a download.
type Download struct {
	Src string // The source URL to find and download images from.
	Dst string // The destination to place the downloaded images at.

	Pattern string         // A shell pattern to filter images.
	Regex   *regexp.Regexp // A regex to filter images.
}

// Start searches for images on the specified website Src and
// downloads matching images to the desired destination Dst.
// Canceling the context will only pause the download and can
// be resumed to proceed downloading where paused at.
func (d *Download) Start(ctx context.Context) error {
	// TODO Prepare destination

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
	// Download images
	return downloadImages(ctx, imageURLs, d.Dst)
}

func newRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}
	req.Header.Set("User-Agent", "Imaget/alpha image downloader")
	return req, nil
}

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

var imageRegex = regexp.MustCompile(`(http(s?):)([/|.|\w|\s|-])*\.(?:jpg|gif|png)`)

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
	// Filter by pattern and/or regex
	for _, s := range b {
		if d.Pattern != "" {
			if match, err := filepath.Match(d.Pattern, s); err == nil || !match {
				continue
			}
		}
		if d.Regex != nil && !d.Regex.MatchString(s) {
			continue
		}
		a = append(a, s)
	}
	b = nil
	return a
}

func downloadImages(ctx context.Context, imageURLs []string, dst string) error {
	if len(imageURLs) == 0 {
		return nil
	}

	bar := progressbar.New(100)
	defer bar.Clear()
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()

	var buf []byte
	for i, url := range imageURLs {
		// Download image to temporary file
		fmt.Printf("(%d/%d) Downloading %s", i+1, len(imageURLs), url)
		startTime := time.Now()
		tmpFilename, err := downloadImage(ctx, url, grab.DefaultClient, bar, t)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error downloading image: %v\n", err)
			continue
		}
		fmt.Printf(" - Took %s\n", time.Since(startTime))

		// Copy file to destination
		// TODO do in parallel
		err = func() error {
			src, err := os.Open(tmpFilename)
			if err != nil {
				return err
			}
			defer src.Close()

			dstFile := filepath.Join(dst, filepath.Base(url))
			dst, err := os.OpenFile(dstFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			defer dst.Close()

			_, err = io.CopyBuffer(dst, src, buf)
			if err != nil {
				err = fmt.Errorf("error copying download (%s) to destination (%s): %w", tmpFilename, dstFile, err)
			}
			return err
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

var tmpDir = filepath.Join(os.TempDir(), "imaget")

func downloadImage(
	ctx context.Context,
	imageURL string,
	c *grab.Client,
	bar *progressbar.ProgressBar,
	t *time.Ticker,
) (tmpFilename string, err error) {
	// Create request
	req, err := grab.NewRequest(filepath.Join(tmpDir, filename(imageURL)), imageURL)
	if err != nil {
		return "", fmt.Errorf("error creating new download request for %q: %w", imageURL, err)
	}
	req = req.WithContext(ctx)

	// Start download
	res := c.Do(req)

	// Download progress
	bar.ChangeMax64(res.Size)
loop:
	for {
		select {
		case <-t.C:
			updateDownloadBar(bar, res.Progress(), res.BytesComplete())
		case <-res.Done:
			break loop
		}
	}
	if res.Err() != nil {
		return "", fmt.Errorf("error download %q: %w", imageURL, res.Err())
	}
	return res.Filename, nil
}

func updateDownloadBar(bar *progressbar.ProgressBar, percentage float64, num int64) {
	desc := fmt.Sprintf("%.2f%s Downloaded...", percentage*100, "%")
	bar.Describe(desc)
	_ = bar.Set64(num)
}

func filename(imageURL string) string {
	return base64.URLEncoding.EncodeToString([]byte(imageURL + filepath.Ext(imageURL)))
}
