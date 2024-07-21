package playback

import (
	"fmt"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/marcus-crane/gunslinger/models"
)

type System interface {
	UpdatePlaybackState(update Update) error
	RefreshCurrentPlayback() error
	GetActivePlayback() ([]FullPlaybackEntry, error)
	GetActivePlaybackBySource(source string) ([]FullPlaybackEntry, error)
	DeactivateBySource(source string) error
	GetHistory(limit int) ([]FullPlaybackEntry, error)
}

type Status string

const (
	StatusPlaying Status = "playing"
	StatusPaused  Status = "paused"
	StatusStopped Status = "stopped"
)

type Category string

// TODO: Normalise gaming into videogame and podcast_episode into podcast(?)
const (
	Episode Category = "episode"
	Gaming  Category = "gaming"
	Manga   Category = "manga"
	Movie   Category = "movie"
	Podcast Category = "podcast_episode"
	Track   Category = "track"
)

type Source string

const (
	Anilist    Source = "anilist"
	Plex       Source = "plex"
	Steam      Source = "steam"
	Trakt      Source = "trakt"
	TraktCasts Source = "traktcasts"
)

// PlaybackEntry is a unique instance of a piece of media being played. If a movie is watched 5 times,
// there will be one MediaItem entry with five unique PlaybackEntry instances. PlaybackEntry instances
// may be "revived" such as if a podcast is paused and then picked up again the next day. Once completed,
// a PlaybackEntry should not be reused though.
type PlaybackEntry struct {
	ID        int       `db:"id"`
	MediaID   string    `db:"media_id"`
	Category  string    `db:"category"`
	CreatedAt time.Time `db:"created_at"`
	Elapsed   int       `db:"elapsed"` // milliseconds
	Status    Status    `db:"status"`
	IsActive  bool      `db:"is_active"`
	UpdatedAt time.Time `db:"updated_at"`
	Source    Source    `db:"source"`
}

// MediaItem stores metadata about each piece of media that is played ie; movies, tv series, games
// It's generic enough to support differences in media types such as music tracks needing a title,
// album name and artist while a game may need a title, developer name and year. Currently, each
// media source scraper is responsible for constructing the appropriate titles such as joining
// a movie name and year into a title field. In future, an explicit author field may be added.
type MediaItem struct {
	ID              string                     `db:"id"`
	Title           string                     `db:"title"`
	Subtitle        string                     `db:"subtitle"`
	Category        string                     `db:"category"`
	Duration        int                        `db:"duration"`
	Source          string                     `db:"source"`
	Image           string                     `db:"image"`
	DominantColours models.SerializableColours `db:"dominant_colours"`
}

// FullPlaybackEntry reflects a single PlaybackEntry with MediaItem metadata attached
// in order to power any clients that want to render full playback info.
type FullPlaybackEntry struct {
	// MediaItem fields
	ID              string                     `db:"id" json:"id"`
	Title           string                     `db:"title" json:"title"`
	Subtitle        string                     `db:"subtitle" json:"subtitle"`
	Category        string                     `db:"category" json:"category"`
	Duration        int                        `db:"duration" json:"duration_ms"` // TODO: Drop _ms suffix
	Source          string                     `db:"source" json:"source"`
	Image           string                     `db:"image" json:"image"` // TODO: Construct image URL from media_id
	DominantColours models.SerializableColours `db:"dominant_colours" json:"dominant_colours"`

	// PlaybackEntry fields
	PlaybackID int       `db:"playback_id" json:"-"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	Elapsed    int       `db:"elapsed" json:"elapsed_ms"` // TODO: Drop _ms suffix
	Status     Status    `db:"status" json:"status"`
	IsActive   bool      `db:"is_active" json:"is_active"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

type Update struct {
	MediaItem MediaItem
	Elapsed   time.Duration
	Status    Status
}

func GenerateMediaID(p *Update) string {
	hashString := fmt.Sprintf("%s-%s-%s-%d-%s-%s",
		p.MediaItem.Title,
		p.MediaItem.Subtitle,
		p.MediaItem.Category,
		p.MediaItem.Duration,
		p.MediaItem.Source,
		p.MediaItem.Image,
	)
	return fmt.Sprintf(
		"%s:%s:%d",
		p.MediaItem.Source,
		p.MediaItem.Category,
		xxhash.Sum64String(hashString),
	)
}
