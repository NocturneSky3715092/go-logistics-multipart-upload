package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"logistics-multipart-upload/infrai"
)

const bucket = "logistics-media"
const partSize = 8 * 1024 * 1024

func main() {
	if len(os.Args) != 2 {
		panic("usage: go run . <media-file>")
	}
	path := os.Args[1]
	client, err := infrai.NewClient()
	must(err)
	// Provisioning is an explicit first step for a new storage account.
	must(client.CreateBucket(bucket))
	key := "incoming/" + filepath.Base(path)
	uploadID, err := client.CreateMultipart(bucket, key)
	must(err)
	file, err := os.Open(path)
	must(err)
	defer file.Close()
	parts := make([]map[string]any, 0)
	partNumber := 1
	for {
		payload := make([]byte, partSize)
		n, readErr := io.ReadFull(file, payload)
		if readErr == io.ErrUnexpectedEOF {
			payload = payload[:n]
		}
		if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
			must(readErr)
		}
		if n > 0 {
			url, err := client.PresignPart(uploadID, partNumber)
			must(err)
			etag, err := infrai.UploadPart(url, payload)
			must(err)
			parts = append(parts, map[string]any{"part_number": partNumber, "etag": etag})
			partNumber++
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
	}
	must(client.CompleteMultipart(uploadID, parts))
	fmt.Printf("uploaded %s in %d parts\n", key, len(parts))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

var _ = strconv.Itoa
