package semantic

// Result is a single search result from the semantic sidecar.
type Result struct {
	PhotoUID       string  `json:"image_id"`
	Score          float64 `json:"score"`
	AestheticScore float64 `json:"aesthetic_score"`
}

// Results is a slice of Result.
type Results []Result
