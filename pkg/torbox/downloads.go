package torbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/robofuse/robofuse/internal/request"
)

// downloads.go handles TorBox download link operations.

// linkExpiryDuration is how long TorBox download links last (~3 hours).
const linkExpiryDuration = 3 * time.Hour

// bonusFolderPatterns contains folder names that indicate bonus/extras content.
// TorBox file paths include the full torrent directory structure, so a file at
// "Zootopia (2016)/Extras/Behind The Scenes.mkv" will be filtered out.
var bonusFolderPatterns = []string{
	"extras",
	"extra",
	"bonus",
	"featurettes",
	"featurette",
	"behind the scenes",
	"deleted scenes",
	"special features",
	"interviews",
	"trailers",
}

// isBonusContent checks if a file path within a torrent is in a bonus/extras folder.
func isBonusContent(filePath string) bool {
	// Normalize separators and lowercase
	filePath = strings.ToLower(filepath.ToSlash(filePath))
	parts := strings.Split(filePath, "/")
	// Check each directory segment (skip the last one which is the filename)
	for _, part := range parts[:len(parts)-1] {
		part = strings.TrimSpace(part)
		for _, pattern := range bonusFolderPatterns {
			if part == pattern {
				return true
			}
		}
	}
	return false
}

// isSampleFile checks if a filename looks like a sample file.
func isSampleFile(filename string) bool {
	name := strings.ToLower(strings.TrimSuffix(filename, filepath.Ext(filename)))
	return name == "sample" || strings.HasPrefix(name, "sample-") || strings.HasPrefix(name, "sample.")
}

// GetDownloads builds a download list from completed torrents and their files.
// Unlike Real-Debrid, TorBox doesn't have a separate "downloads" cache.
// We use the requestdl redirect URL as the download URL — media players hit it
// and get redirected to the actual file. This avoids needing to pre-fetch URLs
// that expire after ~3 hours.
func (c *Client) GetDownloads(torrents []*Torrent) ([]*Download, error) {
	c.logger.Debug().Msg("Building downloads from torrent files...")

	var downloads []*Download
	var skippedBonus, skippedSample int
	for _, t := range torrents {
		for _, f := range t.Files {
			// Filter out bonus/extras content using the full file path
			if f.Name != "" && isBonusContent(f.Name) {
				skippedBonus++
				continue
			}

			// Filter out sample files
			if isSampleFile(f.ShortName) {
				skippedSample++
				continue
			}

			// Build a synthetic link identifier for matching: "torbox://{torrent_id}/{file_id}"
			link := fmt.Sprintf("torbox://%d/%d", t.ID, f.ID)

			// Use the requestdl endpoint with redirect=true as the download URL.
			// When a media player hits this URL, TorBox redirects to the actual file.
			downloadURL := fmt.Sprintf("%s/torrents/requestdl?token=%s&torrent_id=%d&file_id=%d&redirect=true",
				c.Host, c.APIKey, t.ID, f.ID)

			downloads = append(downloads, &Download{
				TorrentID: t.ID,
				FileID:    f.ID,
				Filename:  f.ShortName,
				MimeType:  f.MimeType,
				Filesize:  f.Size,
				Link:      link,
				URL:       downloadURL,
			})
		}
	}

	if skippedBonus > 0 || skippedSample > 0 {
		c.logger.Debug().
			Int("bonus", skippedBonus).
			Int("sample", skippedSample).
			Msg("Filtered out bonus/sample files")
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
