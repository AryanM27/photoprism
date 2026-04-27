package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/photoprism/photoprism/internal/ai/semantic"
	"github.com/photoprism/photoprism/internal/auth/acl"
	"github.com/photoprism/photoprism/internal/form"
	"github.com/photoprism/photoprism/internal/photoprism/get"
	"github.com/photoprism/photoprism/pkg/i18n"
)

const (
	semanticDefaultCount = 20
	semanticMaxCount     = 100
)

// SemanticSearchResult is the response item for semantic search results.
type SemanticSearchResult struct {
	ID             string  `json:"id"`
	Score          float64 `json:"score"`
	AestheticScore float64 `json:"aesthetic_score,omitempty"`
	URL            string  `json:"url"`
}

// SearchPhotosSemantic finds photos using semantic (CLIP) search and returns them as JSON.
//
//	@Summary		finds photos using semantic (CLIP) search and returns them as JSON
//	@Id				SearchPhotosSemantic
//	@Tags			Photos
//	@Produce		json
//	@Success		200				{array}		SemanticSearchResult
//	@Failure		400,401,403,503	{object}	i18n.Response
//	@Param			q				query		string	true	"natural-language search query"
//	@Param			count			query		int		false	"maximum number of results (default 20, max 100)"
//	@Param			rerank			query		bool	false	"rerank results using aesthetic score"
//	@Router			/api/v1/photos/semantic [get]
func SearchPhotosSemantic(router *gin.RouterGroup) {
	router.GET("/photos/semantic", func(c *gin.Context) {
		s := AuthAny(c, acl.ResourcePhotos, acl.Permissions{acl.ActionSearch, acl.ActionView, acl.AccessShared})

		if s.Abort(c) {
			return
		}

		var f form.SearchPhotosSemantic
		if err := c.MustBindWith(&f, binding.Form); err != nil {
			AbortBadRequest(c, err)
			return
		}

		if f.Count <= 0 {
			f.Count = semanticDefaultCount
		} else if f.Count > semanticMaxCount {
			f.Count = semanticMaxCount
		}

		conf := get.Config()
		semConf := conf.SemanticConfig()

		if !semConf.IsEnabled() {
			Abort(c, http.StatusServiceUnavailable, i18n.ErrFeatureDisabled)
			return
		}

		semResults, err := semantic.New(semConf).Search(f.Q, f.Count, f.Rerank, s.UserUID)
		if err != nil {
			Abort(c, http.StatusInternalServerError, i18n.ErrUnexpected)
			return
		}

		results := make([]SemanticSearchResult, 0, len(semResults))
		for _, r := range semResults {
			results = append(results, SemanticSearchResult{
				ID:             r.PhotoUID,
				Score:          r.Score,
				AestheticScore: r.AestheticScore,
				URL:            conf.BaseUri("/api/v1/semantic/image/" + r.PhotoUID),
			})
		}

		AddCountHeader(c, len(results))
		AddLimitHeader(c, f.Count)
		AddOffsetHeader(c, 0)
		AddTokenHeaders(c, s)

		c.JSON(http.StatusOK, results)
	})
}
