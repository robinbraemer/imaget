package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/robinbraemer/imaget"
	"os"
	"regexp"
	"strings"
	"time"
)

const usageMessage = `usage: imaget -u URL [-y] [-d destination] [-t timeout] [-p pattern] [-r regex]

Imaget is a convenient image tool for finding images on any http(s) website and
downloading them with optional flags to tweak tool behaviour and images output.

Flags
-----------------

-u (required): is the http(s) URL to find and images from to download.

-d (optional): is the destination to download the images to.
               It can either be a directory to drop all images into a .zip
               archive to create and save all the images into.

-f (optional): saves the downloaded images as a flat hierarchie,
               instead of creating subdirectories as per the image download URLs.
               The name of the file is the base64 encoded download URL of the image.

-t (optional): is the timeout to wait before pausing the download
               and quitting the programm. Zero or below means no timeout.
               Example: 3m3s

-r (optional): is a regular expression to only download images from matching URLs.
               Example: "(jpg|png)$", "^https?://"

-y (optional): starts the download directly without asking.

-s (optional): will make the console silent and produce no console output.
               If used the -y flag is used automatically.
`

func usage() {
	fmt.Fprintf(os.Stderr, usageMessage)
	os.Exit(2)
}

var (
	u   = flag.String("u", "", "download from this url")
	dst = flag.String("d", ".", "destination to drop the images at")
	_   = flag.Bool("y", false, "accept download")
	_   = flag.Bool("f", false, "save as flat hierarchie")
	_   = flag.Bool("s", false, "disable console output")
	t   = flag.Duration("t", time.Hour, "download timeout")
	r   = flag.String("r", "", "filter images using regex (default: no filter)")
)

func main() {
	flag.Parse()

	if err := Main(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Main programm function
func Main() error {
	// Parse input flags
	download, err := parse()
	if err != nil {
		return err
	}
	// Setup timeout
	ctx := context.Background()
	if *t > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, *t)
		defer cancel()
	}
	// Start download
	return download.Start(ctx)
}

func parse() (d *imaget.Download, err error) {
	if *u == "" {
		usage()
	}
	// Prepend http protocol if missing
	if !strings.HasPrefix(*u, "http") {
		*u = "http://" + *u
	}
	// Compile regex matcher
	var reg *regexp.Regexp
	if *r != "" {
		reg, err = regexp.Compile(*r)
		if err != nil {
			return nil, fmt.Errorf("error compiling regex (-r flag): %w", err)
		}
	}
	// Silent: no console activity
	silent := flagPassed("s")
	if silent {
		imaget.Stdout = &nopWriter{}
		imaget.Stderr = &nopWriter{}
	}
	// Create reusable progress bar for showing downloads
	var pBar imaget.ProgressBar
	if silent {
		pBar = &nopProgressBar{}
	} else {
		const barTpl = pb.ProgressBarTemplate(`{{percent . }} {{bar . }}  {{counters . }} {{speed . }}`)
		pBar = &progressBar{barTpl.New(0).
			Set(pb.Bytes, true).
			SetRefreshRate(10 * time.Millisecond)}
	}
	return &imaget.Download{
		Src:        *u,
		Dst:        *dst,
		Regex:      reg,
		SkipAccept: silent || flagPassed("y"),
		SaveFlat:   flagPassed("f"),
		Bar:        pBar,
	}, nil
}

func flagPassed(name string) (found bool) {
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (n int, err error) {
	return 0, err
}

type progressBar struct{ *pb.ProgressBar }

func (b *progressBar) Start()             { b.ProgressBar.Start() }
func (b *progressBar) Finish()            { b.ProgressBar.Finish() }
func (b *progressBar) SetTotal(i int64)   { b.ProgressBar.SetTotal(i) }
func (b *progressBar) SetCurrent(i int64) { b.ProgressBar.SetCurrent(i) }

type nopProgressBar struct{}

func (b *nopProgressBar) Start()           {}
func (b *nopProgressBar) Finish()          {}
func (b *nopProgressBar) SetTotal(int64)   {}
func (b *nopProgressBar) SetCurrent(int64) {}
