package torbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/robofuse/robofuse/internal/request"
)

// downloads.go handles TorBox download link operations.

// linkExpiryDuration is how long TorBox download links last (~3 hours).
const linkExpiryDuration = 3 * time.Hour

// GetDownloads builds a download list from completed torrents and their files.
// Unlike Real-Debrid, TorBox doesn't have a separate "downloads" cache.
// Instead, we build the download list from torrent files, requesting download URLs on demand.
func (c *Client) GetDownloads(torrents []*Torrent) ([]*Download, error) {
	c.logger.Debug().Msg("Building downloads from torrent files...")

	var downloads []*Download
	for _, t := range torrents {
		for _, f := range t.Files {
			// Build a synthetic link identifier for matching: "torbox://{torrent_id}/{file_id}"
			link := fmt.Sprintf("torbox://%d/%d", t.ID, f.ID)

			downloads = append(downloads, &Download{
				TorrentID: t.ID,
				FileID:    f.ID,
				Filename:  f.ShortName,
				MimeType:  f.MimeType,
				Filesize:  f.Size,
				Link:      link,
			})
		}
	}

	c.logger.Debug().
		Int("count", len(downloads)).
		Msg("Downloads built from torrent files")

	return downloads, nil
}

// Download represents a file available for download from TorBox.
type Download struct {
	TorrentID int
	FileID    int
	Filename  string
	MimeType  string
	Filesize  int64
	Link      string // Synthetic link: "torbox://{torrent_id}/{file_id}"
	URL       string // Actual download URL (populated by RequestDownloadLink)
}

// RequestDownloadLink gets a direct download URL for a specific file.
func (c *Client) RequestDownloadLink(torrentID, fileID int) (string, error) {
	url := fmt.Sprintf("%s/torrents/requestdl?token=%s&torrent_id=%d&file_id=%d",
		c.Host, c.APIKey, torrentID, fileID)

	req, _ := http.NewRequest(http.MethodGet, url, nil)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting download link: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// TorBox wraps the response in the standard API wrapper
	var apiResp APIResponse[string]
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("parsing download link response: %w", err)
	}

	if !apiResp.Success || apiResp.Data == "" {
		return "", fmt.Errorf("no download link in response: %s", apiResp.Detail)
	}

	return apiResp.Data, nil
}

// CheckLink validates if a torrent is still active.
// TorBox doesn't have a direct link-check endpoint, so we verify the torrent still exists.
func (c *Client) CheckLink(torrentID string) error {
	url := fmt.Sprintf("%s/torrents/mylist?id=%s", c.Host, torrentID)
	req, _ := http.NewRequest(http.MethodGet, url, nil)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("checking link: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return request.ErrLinkBroken
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
