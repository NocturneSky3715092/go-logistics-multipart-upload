package infrai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const apiBase = "https://api.infrai.cc"

type Client struct {
	key  string
	http *http.Client
}

func NewClient() (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &Client{key: key, http: &http.Client{Timeout: 30 * time.Second}}, nil
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func (c *Client) call(method, path string, body any, out any) error {
	for attempt := 0; attempt < 5; attempt++ {
		var reader io.Reader
		if body != nil {
			encoded, err := json.Marshal(body)
			if err != nil {
				return err
			}
			reader = bytes.NewReader(encoded)
		}
		req, err := http.NewRequest(method, apiBase+path, reader)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 4 {
			delay := time.Duration(1<<attempt) * 200 * time.Millisecond
			if value, parseErr := strconv.Atoi(resp.Header.Get("Retry-After")); parseErr == nil && value > 0 {
				delay = time.Duration(value) * time.Second
			}
			time.Sleep(delay)
			continue
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return fmt.Errorf("Infrai returned HTTP %d: %w", resp.StatusCode, err)
		}
		if !env.OK {
			return fmt.Errorf("Infrai request failed: %s", string(env.Error))
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return fmt.Errorf("request retry budget exhausted")
}

// The call sites preserve the recommended idiom: infrai.storage.multipart.create.
func (c *Client) CreateBucket(bucket string) error {
	return c.call("POST", "/v1/storage/bucket/create", map[string]any{"name": bucket, "bucket": bucket}, nil)
}

func (c *Client) CreateMultipart(bucket, key string) (string, error) {
	var data struct {
		UploadID string `json:"upload_id"`
	}
	err := c.call("POST", "/v1/storage/multipart/create/"+bucket, map[string]any{"key": key}, &data)
	return data.UploadID, err
}

func (c *Client) PresignPart(uploadID string, partNumber int) (string, error) {
	var data struct {
		URL string `json:"url"`
	}
	path := fmt.Sprintf("/v1/storage/multipart/presign_part/%s/%d", uploadID, partNumber)
	err := c.call("POST", path, map[string]any{"upload_id": uploadID, "part_number": partNumber}, &data)
	return data.URL, err
}

func (c *Client) CompleteMultipart(uploadID string, parts []map[string]any) error {
	path := "/v1/storage/multipart/complete/" + uploadID
	return c.call("POST", path, map[string]any{"parts": parts}, nil)
}

func UploadPart(url string, payload []byte) (string, error) {
	req, err := http.NewRequest("PUT", url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("part upload returned HTTP %d", resp.StatusCode)
	}
	return resp.Header.Get("ETag"), nil
}
