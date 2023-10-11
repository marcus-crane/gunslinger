package shared

type DBMediaItem struct {
	ID         uint   `json:"-"`
	CreatedAt  string `json:"-"`
	OccurredAt int64  `json:"occurred_at"`
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle"`
	Author     string `json:"author"`
	Category   string `json:"category"`
	IsActive   bool   `json:"is_active"`
	Elapsed    int    `json:"elapsed_ms"`
	Duration   int    `json:"duration_ms"`
	Source     string `json:"source"`
	Image      string `json:"image"`
}

type Client interface {
	QueryMediaState() (DBMediaItem, error)
}
