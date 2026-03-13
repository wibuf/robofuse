package torbox

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/robofuse/robofuse/pkg/provider"
)

// provider.go adapts the TorBox Client to the provider.Provider interface.

// Ensure Provider implements provider.Provider at compile time.
var _ provider.Provider = (*Provider)(nil)

// Provider wraps the TorBox Client to implement the provider.Provider interface.
type Provider struct {
	*Client
	// cachedTorrents stores the latest GetTorrents result to avoid duplicate API calls
	// within the same sync cycle (GetDownloads needs the torrent list too).
	cachedTorrents []*Torrent
}

// NewProvider creates a provider.Provider backed by TorBox.
func NewProvider(client *Client) *Provider {
	return &Provider{Client: client}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "torbox"
}

// GetTorrents fetches torrents and returns them as provider types.
// Caches the raw torrent list so GetDownloads() doesn't need a second API call.
func (p *Provider) GetTorrents() ([]*provider.Torrent, []*provider.Torrent, error) {
	downloaded, dead, err := p.Client.GetTorrents()
	if err != nil {
		return nil, nil, err
	}
	// Cache for GetDownloads() to reuse
	p.cachedTorrents = downloaded
	return convertTorrents(downloaded), convertTorrents(dead), nil
}

// GetDownloads fetches all completed torrent files as provider Downloads.
// Reuses cached torrents from GetTorrents() if available.
func (p *Provider) GetDownloads() ([]*provider.Download, error) {
	torrents := p.cachedTorrents
	if torrents == nil {
		// Fallback: fetch if GetTorrents() wasn't called first
		var err error
		torrents, _, err = p.Client.GetTorrents()
		if err != nil {
			return nil, err
		}
	}

	tbDownloads, err := p.Client.GetDownloads(torrents)
	if err != nil {
		return nil, err
	}

	return convertDownloads(tbDownloads), nil
}

// UnrestrictLink generates a direct download URL from a TorBox synthetic link.
// The link format is "torbox://{torrent_id}/{file_id}".
func (p *Provider) UnrestrictLink(link string) (*provider.Download, error) {
	torrentID, fileID, err := parseTorBoxLink(link)
	if err != nil {
		return nil, err
	}

	downloadURL, err := p.Client.RequestDownloadLink(torrentID, fileID)
	if err != nil {
		return nil, err
	}

	// We need the file info for the filename/size. Look it up.
	torrent, err := p.getTorrentByID(torrentID)
	if err != nil {
		return nil, err
	}

	var filename string
	var filesize int64
	var mimetype string
	for _, f := range torrent.Files {
		if f.ID == fileID {
			filename = f.ShortName
			filesize = f.Size
			mimetype = f.MimeType
			break
		}
	}

	now := time.Now()
	return &provider.Download{
		ID:        fmt.Sprintf("%d-%d", torrentID, fileID),
		Filename:  filename,
		MimeType:  mimetype,
		Filesize:  filesize,
		Link:      link,
		Download:  downloadURL,
		Generated: now,
		ExpiresAt: now.Add(linkExpiryDuration),
	}, nil
}

// AddMagnet delegates to the underlying Client.
func (p *Provider) AddMagnet(hash string) (string, error) {
	return p.Client.AddMagnet(hash)
}

// SelectVideoFiles delegates to the underlying Client.
func (p *Provider) SelectVideoFiles(torrentID string) (int, error) {
	return p.Client.SelectVideoFiles(torrentID)
}

// DeleteTorrent delegates to the underlying Client.
func (p *Provider) DeleteTorrent(torrentID string) error {
	return p.Client.DeleteTorrent(torrentID)
}

// CheckLink validates a TorBox synthetic link.
func (p *Provider) CheckLink(link string) error {
	torrentID, _, err := parseTorBoxLink(link)
	if err != nil {
		return err
	}
	return p.Client.CheckLink(fmt.Sprintf("%d", torrentID))
}

// getTorrentByID fetches a single torrent by ID.
func (p *Provider) getTorrentByID(torrentID int) (*Torrent, error) {
	return p.Client.GetTorrentByID(fmt.Sprintf("%d", torrentID))
}

// parseTorBoxLink parses a synthetic TorBox link "torbox://{torrent_id}/{file_id}".
func parseTorBoxLink(link string) (torrentID, fileID int, err error) {
	if !strings.HasPrefix(link, "torbox://") {
		return 0, 0, fmt.Errorf("invalid torbox link: %s", link)
	}

	parts := strings.Split(strings.TrimPrefix(link, "torbox://"), "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid torbox link format: %s", link)
	}

	torrentID, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid torrent ID in link: %s", link)
	}

	fileID, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid file ID in link: %s", link)
	}

	return torrentID, fileID, nil
}

func convertTorrents(torrents []*Torrent) []*provider.Torrent {
	result := make([]*provider.Torrent, len(torrents))
	for i, t := range torrents {
		// Build links from files: each file gets a synthetic link
		links := make([]string, len(t.Files))
		for j, f := range t.Files {
			links[j] = fmt.Sprintf("torbox://%d/%d", t.ID, f.ID)
		}

		status := "downloaded"
		if t.DownloadState == StateError || t.DownloadState == StateDead {
			status = "dead"
		}

		result[i] = &provider.Torrent{
			ID:       fmt.Sprintf("%d", t.ID),
			Filename: t.Name,
			Hash:     t.Hash,
			Bytes:    t.Size,
			Status:   status,
			Progress: t.Progress,
			Added:    t.GetAddedAt(),
			Links:    links,
		}
	}
	return result
}

func convertDownloads(downloads []*Download) []*provider.Download {
	now := time.Now()
	result := make([]*provider.Download, len(downloads))
	for i, d := range downloads {
		result[i] = &provider.Download{
			ID:        fmt.Sprintf("%d-%d", d.TorrentID, d.FileID),
			Filename:  d.Filename,
			MimeType:  d.MimeType,
			Filesize:  d.Filesize,
			Link:      d.Link,
			Download:  d.URL,
			Generated: now,
			ExpiresAt: now.Add(linkExpiryDuration),
		}
	}
	return result
}
