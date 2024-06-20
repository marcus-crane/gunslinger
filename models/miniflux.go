package models

type MinifluxSavedEntry struct {
	EventType string `json:"event_type"`
	Entry     Entry  `json:"entry"`
}

type Entry struct {
	ID          int         `json:"id"`
	UserID      int         `json:"user_id"`
	FeedID      int         `json:"feed_id"`
	Status      string      `json:"status"`
	Hash        string      `json:"hash"`
	URL         string      `json:"url"`
	CommentsURL string      `json:"comments_url"`
	PublishedAt string      `json:"published_at"`
	CreatedAt   string      `json:"created_at"`
	ChangedAt   string      `json:"changed_at"`
	Content     string      `json:"content"`
	Author      string      `json:"author"`
	ShareCode   string      `json:"share_code"`
	Starred     bool        `json:"starred"`
	ReadingTime int         `json:"reading_time"`
	Enclosures  []Enclosure `json:"enclosures"`
	Tags        []string    `json:"tags"`
	Feed        Feed        `json:"feed"`
}

type Feed struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	FeedURL   string `json:"feed_url"`
	SiteURL   string `json:"site_url"`
	Title     string `json:"title"`
	CheckedAt string `json:"checked_at"`
}

type Enclosure struct {
	ID               int    `json:"id"`
	UserID           int    `json:"user_id"`
	EntryID          int    `json:"entry_id"`
	URL              string `json:"url"`
	MimeType         string `json:"mime_type"`
	Size             int64  `json:"size"`
	MediaProgression int    `json:"media_progression"`
}
