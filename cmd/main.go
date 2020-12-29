package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/robinbraemer/imaget"
	"os"
	"regexp"
	"strings"
	"time"
)

const usageMessage = `usage: imaget -u URL [-d destination] [-t timeout] [-p pattern] [-r regex]

Imaget is a convenient image tool for finding images on any http(s) website and
downloading them with optional flags to tweak tool behaviour and images output.

Flags
-----------------

-u (required) is the http(s) URL to download from.

-d (optional): is the destination to download the images to.
  It can either be a directory to drop all images into or
  the resulting archive containing all the images.
  Supported extensions are: .zip | .tar | .tar.gz

-t (optional): is the timeout to wait before pausing the download
  and quitting the programm. Zero or below means no timeout.
  Example: 3m3s

-p (optional): is a shell pattern to only download matching images.
  pattern:
	{ term }
	term:
	'*'         matches any sequence of non-/ characters
	'?'         matches any single non-/ character
	'[' [ '^' ] { character-range } ']'
			  character class (must be non-empty)
	c           matches character c (c != '*', '?', '\\', '[')
	'\\' c      matches character c
	
  character-range:
	c           matches character c (c != '\\', '-', ']')
	'\\' c      matches character c
	lo '-' hi   matches character c for lo <= c <= hi

-r (optional): is a regex expression to only download matching images.
`

func usage() {
	fmt.Fprintf(os.Stderr, usageMessage)
	os.Exit(2)
}

var (
	u   = flag.String("u", "", "download from this url")
	dst = flag.String("d", ".", "destination to drop the images at (default: current directory)")
	t   = flag.Duration("t", time.Hour, "download timeout (default: 1h)")
	p   = flag.String("p", "", "filter images using shell pattern (default: no filter)")
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
	// Compile regex
	var reg *regexp.Regexp
	if *r != "" {
		reg, err = regexp.Compile(*r)
		if err != nil {
			return nil, fmt.Errorf("error compiling regex (-r flag): %w", err)
		}
	}
	return &imaget.Download{
		Src:     *u,
		Dst:     *dst,
		Pattern: *p,
		Regex:   reg,
	}, nil
}
