package form

// SearchPhotosSemantic represents a semantic photo search request.
type SearchPhotosSemantic struct {
	Q      string `form:"q"`
	Count  int    `form:"count"`
	Rerank bool   `form:"rerank"`
}
