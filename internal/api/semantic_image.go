package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/photoprism/photoprism/internal/photoprism/get"
)

// SemanticImage proxies an image from the ML sidecar by image_id.
//
//	@Summary		proxy an image from the semantic sidecar by image_id
//	@Id				SemanticImage
//	@Tags			Photos
//	@Produce		image/jpeg
//	@Success		200	{file}		binary
//	@Failure		404	{object}	i18n.Response
//	@Failure		502	{object}	i18n.Response
//	@Param			id	path		string	true	"image_id (MD5 hash)"
//	@Router			/api/v1/semantic/image/{id} [get]
func SemanticImage(router *gin.RouterGroup) {
	router.GET("/semantic/image/:id", func(c *gin.Context) {
		id := c.Param("id")

		conf := get.Config()
		semConf := conf.SemanticConfig()

		if !semConf.IsEnabled() || semConf.Uri == "" {
			c.Status(http.StatusServiceUnavailable)
			return
		}

		upstreamURL := fmt.Sprintf("%s/image/%s", semConf.Uri, id)
		resp, err := http.Get(upstreamURL) //nolint:gosec
		if err != nil {
			c.Status(http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			c.Status(http.StatusNotFound)
			return
		}

		if resp.StatusCode != http.StatusOK {
			c.Status(http.StatusBadGateway)
			return
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "image/jpeg"
		}

		c.Status(http.StatusOK)
		c.Header("Content-Type", contentType)
		c.Header("Cache-Control", "public, max-age=86400")
		_, _ = io.Copy(c.Writer, resp.Body)
	})
}
