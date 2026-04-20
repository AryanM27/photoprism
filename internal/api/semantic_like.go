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

type semanticLikeRequest struct {
	ImageID string  `json:"image_id"`
	Query   string  `json:"query"`
	Score   float64 `json:"score"`
}

// SemanticLike records a user like for a semantic search result.
//
//	@Summary		record a like for a semantic search result
//	@Id				SemanticLike
//	@Tags			Photos
//	@Accept			json
//	@Success		204
//	@Failure		400,401,503	{object}	i18n.Response
//	@Router			/api/v1/semantic/like [post]
func SemanticLike(router *gin.RouterGroup) {
	router.POST("/semantic/like", func(c *gin.Context) {
		s := AuthAny(c, acl.ResourcePhotos, acl.Permissions{acl.ActionSearch, acl.ActionView, acl.AccessShared})

		if s.Abort(c) {
			return
		}

		var req semanticLikeRequest
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
		upstreamURL := fmt.Sprintf("%s/feedback/like", semConf.Uri)
		resp, err := http.Post(upstreamURL, "application/json", bytes.NewReader(body)) //nolint:gosec
		if err != nil {
			c.Status(http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)

		c.Status(http.StatusNoContent)
	})
}
