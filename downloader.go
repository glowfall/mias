package main

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type cachingDownloader struct {
	cacheDir   string
	httpClient *http.Client
}

func NewCachingDownloader() *cachingDownloader {
	executablePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	cacheDir := filepath.Dir(executablePath) + "/cache"

	log.Printf("Cache dir: %s\n", cacheDir)
	if err := os.MkdirAll(cacheDir, 0777); err != nil {
		panic(err)
	}

	return &cachingDownloader{
		cacheDir: cacheDir,
		httpClient: &http.Client{
			Transport: &http.Transport{
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			Timeout: time.Minute,
		},
	}
}

func (d *cachingDownloader) DownloadOrGetCached(link string) (string, error) {
	cachedPath := d.cacheDir + "/" + getLinkHash(link)

	body, err := os.ReadFile(cachedPath)
	if err == nil {
		fmt.Printf("found in cache: %s\n", link)
		return string(body), nil
	}

	fmt.Printf("downloading: %s\n", link)

	r, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return "", err
	}
	r.Header.Add("Referer", "https://www.cuenation.com/?page=cues&folder=asot")

	resp, err := d.httpClient.Do(r)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(cachedPath, body, 0777); err != nil {
		fmt.Printf("unable to write to cache %s: %+v\n", cachedPath, err)
	}

	return string(body), nil
}

func getLinkHash(link string) string {
	sha := sha1.New()
	sha.Write([]byte(link))
	return hex.EncodeToString(sha.Sum(nil))
}
