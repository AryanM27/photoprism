package semantic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotifyUpload(t *testing.T) {
	t.Run("PostsCorrectBody", func(t *testing.T) {
		var gotPath, gotUID string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/upload/notify", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var body notifyUploadRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
			}
			gotUID = body.UserID
			gotPath = body.StagingPath
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		New(Config{Enabled: true, Uri: srv.URL}).NotifyUpload("u123", "/photoprism/storage/users/u123/token")

		assert.Equal(t, "u123", gotUID)
		assert.Equal(t, "/photoprism/storage/users/u123/token", gotPath)
	})

	t.Run("SwallowsServerError", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		// Must not panic or return an error.
		New(Config{Enabled: true, Uri: srv.URL}).NotifyUpload("u1", "/tmp/staging")
	})

	t.Run("SwallowsNetworkError", func(t *testing.T) {
		// Point at a port with nothing listening.
		New(Config{Enabled: true, Uri: "http://127.0.0.1:1"}).NotifyUpload("u1", "/tmp/staging")
	})

	t.Run("SkipsInvalidURI", func(t *testing.T) {
		// validateURI should reject this before any HTTP call is made.
		New(Config{Enabled: true, Uri: "not-a-url"}).NotifyUpload("u1", "/tmp/staging")
	})
}
