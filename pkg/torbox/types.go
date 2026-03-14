package torbox

import (
	"time"
)

// types.go contains TorBox API models.

// APIResponse is the standard TorBox API response wrapper.
type APIResponse[T any] struct {
	Success bool   `json:"success"`
	Detail  string `json:"detail"`
	Error   string `json:"error"`
	Data    T      `json:"data"`
}

// Torrent represents a torrent in TorBox.
type Torrent struct {
	ID               int           `json:"id"`
	Hash             string        `json:"hash"`
	CreatedAt        string        `json:"created_at"`
	UpdatedAt        string        `json:"updated_at"`
	Magnet           string        `json:"magnet"`
	Size             int64         `json:"size"`
	Active           bool          `json:"active"`
	DownloadState    string        `json:"download_state"`
	Seeds            int           `json:"seeds"`
	Peers            int           `json:"peers"`
	Progress         float64       `json:"progress"`
	DownloadSpeed    int           `json:"download_speed"`
	UploadSpeed      int           `json:"upload_speed"`
	Name             string        `json:"name"`
	ETA              int           `json:"eta"`
	ExpiresAt        string        `json:"expires_at"`
	DownloadPresent  bool          `json:"download_present"`
	DownloadFinished bool          `json:"download_finished"`
	Files            []TorrentFile `json:"files"`
	InactiveCheck    int           `json:"inactive_check"`
	Availability     float64       `json:"availability"`
}

// GetAddedAt parses the created_at timestamp.
func (t *Torrent) GetAddedAt() time.Time {
	parsed, _ := time.Parse(time.RFC3339, t.CreatedAt)
	return parsed
}

// TorrentFile represents a file within a TorBox torrent.
type TorrentFile struct {
	ID        int    `json:"id"`
	MD5       string `json:"md5"`
	S3Path    string `json:"s3_path"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mimetype"`
	ShortName string `json:"short_name"`
}

// CreateTorrentResponse is the response from POST /torrents/createtorrent.
type CreateTorrentResponse struct {
	TorrentID int    `json:"torrent_id"`
	Name      string `json:"name"`
	Hash      string `json:"hash"`
	AuthID    string `json:"auth_id"`
}

// ControlTorrentRequest is the request body for POST /torrents/controltorrent.
type ControlTorrentRequest struct {
	TorrentID int    `json:"torrent_id"`
	Operation string `json:"operation"` // "delete", "pause", "resume", "reannounce"
}

// RequestDownloadLinkResponse is the response from GET /torrents/requestdl.
type RequestDownloadLinkResponse struct {
	Link string `json:"data"`
}

// TorBox download state constants.
const (
	StateDownloading = "downloading"
	StateUploading   = "uploading"
	StatePaused      = "paused"
	StateCompleted   = "completed"
	StateCached      = "cached"
	StateMetaDL      = "metaDL"
	StateError       = "error"
	StateDead        = "dead"
)
