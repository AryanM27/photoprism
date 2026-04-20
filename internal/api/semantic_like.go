package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/photoprism/photoprism/internal/auth/acl"
	"github.com/photoprism/photoprism/internal/photoprism/get"
)

type semanticFeedbackRequest struct {
	ImageID string  `json:"image_id"`
	Query   string  `json:"query"`
	Score   float64 `json:"score"`
}

func semanticFeedbackProxy(c *gin.Context, upstreamPath string) {
	s := AuthAny(c, acl.ResourcePhotos, acl.Permissions{acl.ActionSearch, acl.ActionView, acl.AccessShared})
	if s.Abort(c) {
		return
	}

	var req semanticFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortBadRequest(c, err)
		return
	}

	conf := get.Config()
	semConf := conf.SemanticConfig()

	if !semConf.IsEnabled() || semConf.Uri == "" {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	body, _ := json.Marshal(req)
	upstreamURL := fmt.Sprintf("%s/%s", semConf.Uri, upstreamPath)
	resp, err := http.Post(upstreamURL, "application/json", bytes.NewReader(body)) //nolint:gosec
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SemanticLike records a user like for a semantic search result.
//
//	@Summary		record a like for a semantic search result
//	@Id				SemanticLike
//	@Tags			Photos
//	@Router			/api/v1/semantic/like [post]
func SemanticLike(router *gin.RouterGroup) {
	router.POST("/semantic/like", func(c *gin.Context) {
		semanticFeedbackProxy(c, "feedback/like")
	})
}

// SemanticClick records a user click on a semantic search result.
//
//	@Summary		record a click for a semantic search result
//	@Id				SemanticClick
//	@Tags			Photos
//	@Router			/api/v1/semantic/click [post]
func SemanticClick(router *gin.RouterGroup) {
	router.POST("/semantic/click", func(c *gin.Context) {
		semanticFeedbackProxy(c, "feedback/click")
	})
}
