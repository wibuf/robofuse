package provider

import (
	"time"
)

// provider.go defines the common interface for debrid providers (Real-Debrid, TorBox, etc.).

// Torrent represents a torrent from any debrid provider.
type Torrent struct {
	ID       string
	Filename string
	Hash     string
	Bytes    int64
	Status   string // "downloaded", "dead", etc.
	Progress float64
	Added    time.Time
	Ended    time.Time
	Links    []string // Provider-specific link identifiers
	Files    []File
}

// File represents a file within a torrent.
type File struct {
	ID       int
	Path     string
	Bytes    int64
	Selected int
}

// Download represents a cached/unrestricted download with a streamable URL.
type Download struct {
	ID        string
	Filename  string
	MimeType  string
	Filesize  int64
	Link      string    // Original link identifier (for matching)
	Download  string    // Direct download URL (goes inside .strm)
	Generated time.Time // When the download link was generated
	ExpiresAt time.Time // When the download link expires
}

// IsExpired returns true if the download link is expired.
func (d *Download) IsExpired() bool {
	return time.Now().After(d.ExpiresAt)
}

// WillExpireBefore returns true if the link will expire before the given time.
func (d *Download) WillExpireBefore(t time.Time) bool {
	return d.ExpiresAt.Before(t)
}

// STRMCandidate represents a candidate for STRM file generation.
type STRMCandidate struct {
	TorrentID     string
	TorrentFolder string // Name of folder (from torrent filename)
	Filename      string // Name of file (from download filename)
	DownloadURL   string // Direct download URL (goes inside .strm file)
	Link          string // Original link identifier (for matching)
	Filesize      int64
}

// Provider is the interface that debrid providers must implement.
type Provider interface {
	// Name returns the provider name (e.g. "real-debrid", "torbox").
	Name() string

	// GetTorrents fetches all torrents, returning downloaded and dead lists.
	GetTorrents() ([]*Torrent, []*Torrent, error)

	// GetDownloads fetches all cached downloads (deduplicated).
	GetDownloads() ([]*Download, error)

	// UnrestrictLink generates a direct download URL from a provider link.
	// For Real-Debrid this calls /unrestrict/link.
	// For TorBox this calls /torrents/requestdl.
	UnrestrictLink(link string) (*Download, error)

	// AddMagnet adds a torrent by its info hash. Returns the new torrent ID.
	AddMagnet(hash string) (string, error)

	// SelectVideoFiles selects video files in a torrent for downloading.
	// Returns the number of files selected.
	SelectVideoFiles(torrentID string) (int, error)

	// DeleteTorrent removes a torrent from the provider.
	DeleteTorrent(torrentID string) error

	// CheckLink validates if a link is still accessible.
	CheckLink(link string) error
}
