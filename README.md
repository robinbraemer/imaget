# imaget

Imaget is a convenient image tool for finding images on any http(s) website and
downloading them with optional flags to tweak tool behaviour and images output.

## Install

```shell
git clone https://github.com/robinbraemer/imaget.git
cd imaget
go install cmd/
```

## Showcase Video

<p align="center">
    <a href="https://player.vimeo.com/video/496021582">
        <img
            src="https://raw.githubusercontent.com/robinbraemer/imaget/main/etc/play.gif"
            width="400" height="300">
        <br>
    </a>
</p>

## Usage

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