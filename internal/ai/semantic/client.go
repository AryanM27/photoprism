package semantic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/photoprism/photoprism/internal/event"
	"github.com/photoprism/photoprism/pkg/http/safe"
)

var log = event.Log

// Client is an HTTP client for the semantic search sidecar API.
type Client struct {
	conf   Config
	client *http.Client
}

// New returns a new semantic search Client.
func New(conf Config) *Client {
	return &Client{
		conf:   conf,
		client: &http.Client{Timeout: ServiceTimeout},
	}
}

// validateURI returns an error if the configured URI is not a valid HTTP(S) URL.
func (c *Client) validateURI() error {
	_, err := safe.URL(c.conf.Uri)
	return err
}

// Health checks if the sidecar is reachable.
func (c *Client) Health() error {
	if err := c.validateURI(); err != nil {
		return err
	}

	resp, err := c.client.Get(c.conf.Uri + "/health")
	if err != nil {
		return fmt.Errorf("semantic: health check failed (%w)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("semantic: health check returned status %d", resp.StatusCode)
	}

	return nil
}

// Index sends a photo thumbnail to the sidecar for CLIP embedding.
// photoUID is stored as the vector ID in Qdrant.
// thumbPath is the local filesystem path to the tile_224 thumbnail.
func (c *Client) Index(photoUID, thumbPath string) error {
	if err := c.validateURI(); err != nil {
		return err
	}

	if !c.conf.IsEnabled() {
		return nil
	}

	f, err := os.Open(thumbPath)
	if err != nil {
		return fmt.Errorf("semantic: cannot open thumbnail %s (%w)", thumbPath, err)
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// Add image_id field (we use photo_uid as the stable ID in Qdrant).
	if err = w.WriteField("image_id", photoUID); err != nil {
		return fmt.Errorf("semantic: cannot write image_id field (%w)", err)
	}

	// Add file field.
	part, err := w.CreateFormFile("file", filepath.Base(thumbPath))
	if err != nil {
		return fmt.Errorf("semantic: cannot create form file (%w)", err)
	}
	if _, err = io.Copy(part, f); err != nil {
		return fmt.Errorf("semantic: cannot copy thumbnail (%w)", err)
	}
	_ = w.Close()

	req, err := http.NewRequest(http.MethodPost, c.conf.Uri+"/index", &buf)
	if err != nil {
		return fmt.Errorf("semantic: cannot create index request (%w)", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("semantic: index request failed (%w)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("semantic: index returned status %d", resp.StatusCode)
	}

	log.Debugf("semantic: indexed photo %s", photoUID)
	return nil
}

// notifyUploadRequest is the JSON body for the /upload/notify endpoint.
type notifyUploadRequest struct {
	UserID      string `json:"user_id"`
	StagingPath string `json:"staging_path"`
}

// NotifyUpload tells the semantic sidecar about a new user upload so it can
// ingest the staged files into the ML pipeline (S3 → Postgres → validation →
// embedding) before PhotoPrism moves them out of the staging directory.
//
// Errors are logged and swallowed — pipeline failures must never break uploads.
func (c *Client) NotifyUpload(userUID, stagingPath string) {
	if err := c.validateURI(); err != nil {
		log.Debugf("semantic: notify upload skipped, invalid URI: %s", err)
		return
	}

	body, err := json.Marshal(notifyUploadRequest{UserID: userUID, StagingPath: stagingPath})
	if err != nil {
		log.Warnf("semantic: notify upload marshal failed: %s", err)
		return
	}

	// Use a longer timeout than the default ServiceTimeout because the sidecar
	// uploads files to S3 synchronously before returning.
	notifyClient := &http.Client{Timeout: 90 * time.Second}
	resp, err := notifyClient.Post(c.conf.Uri+"/upload/notify", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Warnf("semantic: notify upload request failed: %s", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		log.Warnf("semantic: notify upload returned status %d for user %s", resp.StatusCode, userUID)
		return
	}

	log.Debugf("semantic: notified upload pipeline for user %s (staging: %s)", userUID, stagingPath)
}

// searchRequest is the JSON body for the /search endpoint.
type searchRequest struct {
	Query  string `json:"query"`
	TopK   int    `json:"top_k"`
	Rerank bool   `json:"rerank"`
	UserID string `json:"user_id,omitempty"`
}

// Search queries the sidecar for photos matching the text query.
// Pass a non-empty userUID to restrict results to that user's uploads only.
func (c *Client) Search(query string, topK int, rerank bool, userUID string) (Results, error) {
	if err := c.validateURI(); err != nil {
		return nil, err
	}

	if !c.conf.IsEnabled() {
		return nil, nil
	}

	body, err := json.Marshal(searchRequest{Query: query, TopK: topK, Rerank: rerank, UserID: userUID})
	if err != nil {
		return nil, fmt.Errorf("semantic: cannot marshal search request (%w)", err)
	}

	resp, err := c.client.Post(c.conf.Uri+"/search", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("semantic: search request failed (%w)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("semantic: search returned status %d", resp.StatusCode)
	}

	var results Results
	if err = json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&results); err != nil {
		return nil, fmt.Errorf("semantic: cannot decode search response (%w)", err)
	}

	return results, nil
}
