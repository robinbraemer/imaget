# imaget

Imaget is a convenient image tool for finding images on any http(s) website and downloading them with optional flags to
tweak tool behaviour and images output.

Imaget makes sure to only download the necessary bytes. Therefore,
remembers image URLs and caches the downloads in the
temporary directory of your operating system
(`/tmp/imaget/` for linux) to skip already downloaded images and
also auto-resume incomplete downloads when canceled in
the past.

The tool is user-friendly and can also be used in automation since the `-y` (skip user accept screen) or `-t` (timeout
download)
and other flags are provided for convenience of use.

## Install

No worries, installation is straight forward!

Requirements:

- ([Git](https://git-scm.com/downloads) & [Go](https://golang.org/doc/install))
  **OR** ([Docker](https://docs.docker.com/get-docker/)) installed

**Install with Go:**

```shell
git clone https://github.com/robinbraemer/imaget.git
cd imaget
go install cmd/imaget.go
```

**Test with Docker:**

```shell
docker run -it --rm golang:1.15
git clone https://github.com/robinbraemer/imaget.git
cd imaget
go install cmd/imaget.go
```

Now try running `imaget` in your command line!

## Showcase

While there are many more use cases for this tool this video shows 3 sample commands and how much faster it is due to
the smart cache functionality used when running the last command twice.

1. Download Google's current image above the search box.
    - `imaget -s -f -u google.com`
2. Download all images found on amazon.com to new Zip archive.
    - `imaget -y -f -u amazon.com -d amazon-images.zip`
3. Download all images found on alibaba.com to hierarchical directories.
    - `imaget -y -u alibaba.com -d alibaba-images`
4. Re-run 3. to see how much faster the download is thanks to automatic cache use.
    - `imaget -y -u alibaba.com -d alibaba-images`

<p align="center">
    <a href="https://player.vimeo.com/video/496021582">
        <img alt="Showcase" width="500" height="400"
            src="https://raw.githubusercontent.com/robinbraemer/imaget/main/etc/play.gif">
    </a>
</p>

## Usage

This is what's outputted when typing `imaget` in your command line.

```shell
usage: imaget -u URL [-d destination] [-t timeout] [-r regex] [-y] [-s] [-f]

Imaget is a convenient image tool for finding images on any http(s) website and
downloading them with optional flags to tweak tool behaviour and images output.

Flags
-----------------

-u (required): is the http(s) URL to find and images from to download.

-d (optional): is the destination to download the images to.
               It can either be the directory to save all images at or
               a path to create a .zip archive to save the images in.

-f (optional): saves the downloaded images as a flat hierarchie,
               instead of creating subdirectories as per the image download URLs.
               The name of the file is the base64 encoded download URL of the image.

-t (optional): is the timeout to wait before pausing the download
               and quitting the programm. Zero or below means no timeout.
               Example: 3m3s

-r (optional): is a regular expression to only download images from matching URLs.
               Examples: "(jpg|png)$", "^https?://"

-y (optional): starts the download directly without asking.

-s (optional): will make the console silent and produce no console output.
               If used the -y flag is used automatically.

Example commands
-----------------

Silently download Google's current image above the search box to the current directory.
> imaget -s -f -u google.com	

Download all images on amazon.com to new Zip archive in the current directory.
> imaget -y -f -u amazon.com -d amazon-images.zip

Download all images on alibaba.com to new directory 'alibaba-images' hierarchically sorted by image URL.
> imaget -y -u alibaba.com -d alibaba-images
```

## Extend

The current features of the tool provide a sufficient foundation to other developers to extend functionality and support
more advanced use cases such as...

- Log into Instagram/Pinterest/Facebook/... and download a complete image history
- Build out an image web crawler to automatically find **baby groots** in the wild ;)
    - allow many, many crawlers using a database for caching and distributed coordination

_Which button was it again?_ (╯°□°）╯︵ ┻━┻
![Baby Groot](https://i.pinimg.com/564x/c0/05/2c/c0052c5d1500ba20b64cfb89e188cf9c.jpg)
