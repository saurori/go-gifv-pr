# GIFV (video) -> GIF -> imgur Converter
Command line tool to convert .gifv or any video file to an optimized .gif and upload to imgur. The primary motivation for this tool is for converting existing .gifv links to .gif format for use in GitHub pull requests.

## Configuration
If you plan on uploading converted images to imgur, you must generate a Client ID [here](https://api.imgur.com/oauth2/addclient).

## Usage
```
go-gif-pr -i http://i.imgur.com/some_file.gifv
```

```
go-gif-pr -i /path/to/some_file.gifv
```

## Options
```
 -i  URL or path of the .gifv or video to convert
 -w  Width of the final converted image. Defaults to 300.
 -c  Imgur Client ID. Defaults to ENV var IMGUR_CLIENT_ID.
     If no ID is provided, the result image will be left locally.
 -k  Option to keep intermediary files created during conversion.
```

## Dependencies
### Mac
```
brew install ffmpeg
brew install gifsicle
```

### Linux
```
apt-get install ffmpeg
apt-get install gifsicle
```

## Building
Install [Go](https://golang.org/dl/)
```
go build
```