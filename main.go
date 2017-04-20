package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

type converter struct {
	keepFiles      bool
	outputMarkdown bool
	imageWidth     string
	clientID       string

	startImage    string
	fileToConvert string
	outputImage   string
	endImage      string
}

type imgurResponse struct {
	Success bool
	Data    struct {
		Link string
		Err  string `json:"error"`
	}
}

const (
	tempFileName     = "temp_file_to_convert"
	outputFileName   = "output"
	imgurAPIEndpoint = "https://api.imgur.com/3/image"
)

func main() {
	var conv converter

	flag.StringVar(&conv.startImage, "i", "", "URL or path of the .gifv or video to convert")
	flag.StringVar(&conv.imageWidth, "w", "300", "Width of the final converted image. Defaults to 300.")
	flag.StringVar(&conv.clientID, "c", os.Getenv("IMGUR_CLIENT_ID"), "Imgur Client ID. Defaults to ENV var IMGUR_CLIENT_ID")
	flag.BoolVar(&conv.keepFiles, "k", false, "Option to keep intermediary files created during conversion.")
	flag.BoolVar(&conv.outputMarkdown, "m", false, "Output Markdown formatted text for quick copy/paste.")
	flag.Parse()

	err := conv.validate()
	if err != nil {
		fmt.Println(err)
		return
	}

	defer conv.cleanup()

	err = conv.fetchFile()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = conv.convert()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = conv.upload()
	if err != nil {
		fmt.Println(err)
		return
	}

	if conv.outputMarkdown {
		fmt.Printf("![](%s)\n", conv.endImage)
	} else {
		fmt.Println(conv.endImage)
	}
}

func (c *converter) validate() error {
	if strings.TrimSpace(c.startImage) == "" {
		return errors.New("You must provide an input URL or path")
	}

	return nil
}

func (c *converter) cleanup() {
	if c.keepFiles {
		return
	}

	// Gather files to remove
	var filesToRemove []string

	// Remove downloaded file
	if c.startImage != c.fileToConvert {
		filesToRemove = append(filesToRemove, c.fileToConvert)
	}

	// If file was not uploaded to imgur, leave local copy
	if strings.TrimSpace(c.clientID) != "" {
		filesToRemove = append(filesToRemove, c.outputImage)
	}

	for _, f := range filesToRemove {
		err := os.Remove(f)
		if err != nil {
			fmt.Println("Could not remove file: ", c.fileToConvert)
		}
	}
}

func (c *converter) fetchFile() error {
	// Download the file if remote
	if strings.HasPrefix(c.startImage, "http") {
		err := c.fetchRemote()
		return err
	}

	c.fileToConvert = c.startImage

	// Check if the file exists
	if _, err := os.Stat(c.fileToConvert); os.IsNotExist(err) {
		return errors.New("Input file does not exist")
	}

	return nil
}

func (c *converter) fetchRemote() error {
	url, err := url.Parse(c.startImage)
	if err != nil {
		return err
	}

	fileExt := path.Ext(url.Path)
	// Gifv is a container for mp4
	if fileExt == ".gifv" {
		fileExt = ".mp4"
		c.startImage = strings.Replace(c.startImage, ".gifv", ".mp4", -1)
	}
	c.fileToConvert = tempFileName + fileExt
	temp, err := os.Create(c.fileToConvert)
	defer temp.Close()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", c.startImage, nil)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(temp, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (c *converter) convert() error {
	// Convert movie to gif
	c.outputImage = outputFileName + ".gif"
	ffmpeg := exec.Command("ffmpeg", "-i", c.fileToConvert, "-pix_fmt", "rgb24", "-vf", "scale="+c.imageWidth+":-1", "-f", "gif", c.outputImage)

	var ffmpegErr bytes.Buffer
	ffmpeg.Stderr = &ffmpegErr

	err := ffmpeg.Run()
	if err != nil {
		return errors.New(fmt.Sprint(err) + ": " + ffmpegErr.String())
	}

	// Optimize gif
	sickle := exec.Command("gifsicle", "--careful", "-O3", "--batch", c.outputImage)

	var sicklekErr bytes.Buffer
	sickle.Stderr = &sicklekErr

	err = sickle.Run()
	if err != nil {
		return errors.New(fmt.Sprint(err) + ": " + sicklekErr.String())
	}

	return nil
}

func (c *converter) upload() error {
	clientID := strings.TrimSpace(c.clientID)
	if clientID == "" {
		fmt.Println("No imgur Client ID provided. File will be retained locally.")
		c.endImage = c.outputImage
		return nil
	}

	// Prepare multi-part body
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	f, err := os.Open(c.outputImage)
	if err != nil {
		return err
	}
	defer f.Close()
	fw, err := w.CreateFormFile("image", c.outputImage)
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, f); err != nil {
		return err
	}
	w.Close()

	req, err := http.NewRequest("POST", imgurAPIEndpoint, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Client-ID "+c.clientID)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var imgur imgurResponse
	err = json.NewDecoder(resp.Body).Decode(&imgur)
	if err != nil {
		return err
	}

	if imgur.Success {
		c.endImage = imgur.Data.Link
	} else {
		return errors.New("imgur error: " + imgur.Data.Err)
	}

	return nil
}
