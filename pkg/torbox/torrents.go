package torbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

// torrents.go handles TorBox torrent operations.

// GetTorrents fetches all torrents from TorBox.
// Returns: (downloaded, dead, error)
func (c *Client) GetTorrents() ([]*Torrent, []*Torrent, error) {
	c.logger.Debug().Msg("Fetching all torrents...")

	var allTorrents []*Torrent
	offset := 0
	limit := 1000

	for {
		url := fmt.Sprintf("%s/torrents/mylist?limit=%d&offset=%d", c.Host, limit, offset)
		req, _ := http.NewRequest(http.MethodGet, url, nil)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, nil, fmt.Errorf("fetching torrents: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
		}

		var apiResp APIResponse[[]*Torrent]
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return nil, nil, fmt.Errorf("parsing torrents: %w", err)
		}

		if !apiResp.Success || apiResp.Data == nil {
			break
		}

		allTorrents = append(allTorrents, apiResp.Data...)
		c.logger.Debug().
			Int("offset", offset).
			Int("count", len(apiResp.Data)).
			Int("total", len(allTorrents)).
			Msg("Fetched torrents batch")

		if len(apiResp.Data) < limit {
			break
		}
		offset += len(apiResp.Data)
	}

	// Classify by download state
	var downloaded []*Torrent
	var dead []*Torrent

	for _, t := range allTorrents {
		switch t.DownloadState {
		case StateCompleted, StateCached, StateUploading:
			downloaded = append(downloaded, t)
		case StateError, StateDead:
			dead = append(dead, t)
		}
	}

	c.logger.Debug().
		Int("total", len(allTorrents)).
		Int("downloaded", len(downloaded)).
		Int("dead", len(dead)).
		Msg("Torrents fetched and filtered")

	return downloaded, dead, nil
}

// AddMagnet adds a magnet link to TorBox. Returns the new torrent ID as a string.
func (c *Client) AddMagnet(hash string) (string, error) {
	magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s", hash)

	url := fmt.Sprintf("%s/torrents/createtorrent", c.Host)

	// TorBox expects multipart/form-data
	body := strings.NewReader(fmt.Sprintf("magnet=%s", magnet))
	req, _ := http.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("adding magnet: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var apiResp APIResponse[CreateTorrentResponse]
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if !apiResp.Success {
		return "", fmt.Errorf("TorBox error: %s", apiResp.Detail)
	}

	id := fmt.Sprintf("%d", apiResp.Data.TorrentID)
	c.logger.Info().Str("id", id).Str("hash", hash[:8]).Msg("Added magnet")
	return id, nil
}

// SelectVideoFiles selects video files from a torrent.
// TorBox doesn't have a file selection step like Real-Debrid — files are available immediately.
// This returns the count of video files in the torrent.
func (c *Client) SelectVideoFiles(torrentID string) (int, error) {
	// Fetch torrent info to count video files
	url := fmt.Sprintf("%s/torrents/mylist?id=%s", c.Host, torrentID)
	req, _ := http.NewRequest(http.MethodGet, url, nil)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetching torrent info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	var apiResp APIResponse[*Torrent]
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return 0, fmt.Errorf("parsing torrent info: %w", err)
	}

	if !apiResp.Success || apiResp.Data == nil {
		return 0, fmt.Errorf("torrent not found: %s", torrentID)
	}

	count := 0
	minSize := c.config.MinFileSizeBytes()
	for _, f := range apiResp.Data.Files {
		ext := strings.ToLower(filepath.Ext(f.Name))
		if (ext == ".mkv" || ext == ".mp4") && f.Size >= minSize {
			count++
		}
	}

	return count, nil
}

// DeleteTorrent deletes a torrent from TorBox.
func (c *Client) DeleteTorrent(torrentID string) error {
	url := fmt.Sprintf("%s/torrents/controltorrent", c.Host)

	payload := ControlTorrentRequest{
		Operation: "delete",
	}
	// Parse torrent ID
	fmt.Sscanf(torrentID, "%d", &payload.TorrentID)

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(string(payloadBytes)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deleting torrent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	c.logger.Info().Str("id", torrentID).Msg("Deleted torrent")
	return nil
}
