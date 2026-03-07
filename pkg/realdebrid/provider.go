package realdebrid

import (
	"time"

	"github.com/robofuse/robofuse/pkg/provider"
)

// provider.go adapts the Real-Debrid Client to the provider.Provider interface.

const linkExpiryDuration = 7 * 24 * time.Hour

// Ensure Provider implements provider.Provider at compile time.
var _ provider.Provider = (*Provider)(nil)

// Provider wraps the Real-Debrid Client to implement the provider.Provider interface.
type Provider struct {
	*Client
}

// NewProvider creates a provider.Provider backed by Real-Debrid.
func NewProvider(client *Client) *Provider {
	return &Provider{Client: client}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "real-debrid"
}

// GetTorrents fetches torrents and returns them as provider types.
func (p *Provider) GetTorrents() ([]*provider.Torrent, []*provider.Torrent, error) {
	downloaded, dead, err := p.Client.GetTorrents()
	if err != nil {
		return nil, nil, err
	}
	return convertTorrents(downloaded), convertTorrents(dead), nil
}

// GetDownloads fetches downloads and returns them as provider types.
func (p *Provider) GetDownloads() ([]*provider.Download, error) {
	downloads, err := p.Client.GetDownloads()
	if err != nil {
		return nil, err
	}
	return convertDownloads(downloads), nil
}

// UnrestrictLink unrestricts a link and returns a provider Download.
func (p *Provider) UnrestrictLink(link string) (*provider.Download, error) {
	dl, err := p.Client.UnrestrictLink(link)
	if err != nil {
		return nil, err
	}
	return convertDownload(dl), nil
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

// CheckLink delegates to the underlying Client.
func (p *Provider) CheckLink(link string) error {
	return p.Client.CheckLink(link)
}

func convertTorrents(torrents []*Torrent) []*provider.Torrent {
	result := make([]*provider.Torrent, len(torrents))
	for i, t := range torrents {
		result[i] = &provider.Torrent{
			ID:       t.ID,
			Filename: t.Filename,
			Hash:     t.Hash,
			Bytes:    t.Bytes,
			Status:   t.Status,
			Progress: t.Progress,
			Added:    t.Added,
			Ended:    t.Ended,
			Links:    t.Links,
		}
	}
	return result
}

func convertDownload(d *Download) *provider.Download {
	return &provider.Download{
		ID:        d.ID,
		Filename:  d.Filename,
		MimeType:  d.MimeType,
		Filesize:  d.Filesize,
		Link:      d.Link,
		Download:  d.Download,
		Generated: d.Generated,
		ExpiresAt: d.Generated.Add(linkExpiryDuration),
	}
}

func convertDownloads(downloads []*Download) []*provider.Download {
	result := make([]*provider.Download, len(downloads))
	for i, d := range downloads {
		result[i] = convertDownload(d)
	}
	return result
}
