package repair

import (
	"fmt"

	"github.com/robofuse/robofuse/internal/config"
	"github.com/robofuse/robofuse/internal/logger"
	"github.com/robofuse/robofuse/pkg/provider"
	"github.com/rs/zerolog"
)

// repair.go re-adds dead torrents using cached magnet data when possible.

// Service handles torrent repair operations
type Service struct {
	p      provider.Provider
	config *config.Config
	logger zerolog.Logger
}

// NewWithProvider creates a new repair service using the provider interface.
func NewWithProvider(p provider.Provider, cfg *config.Config) *Service {
	return &Service{
		p:      p,
		config: cfg,
		logger: logger.New("repair"),
	}
}

// RepairTorrent attempts to repair a dead/failed torrent by reinserting via magnet
func (s *Service) RepairTorrent(torrent *provider.Torrent, dryRun bool) error {
	s.logger.Info().
		Str("id", torrent.ID).
		Str("filename", torrent.Filename).
		Str("hash", torrent.Hash[:8]).
		Msg("Repairing torrent")

	if dryRun {
		s.logger.Info().Msg("[DRY-RUN] Would repair torrent")
		return nil
	}

	// Step 1: Add magnet
	newID, err := s.p.AddMagnet(torrent.Hash)
	if err != nil {
		return fmt.Errorf("adding magnet: %w", err)
	}
	s.logger.Debug().Str("newId", newID).Msg("Added magnet for repair")

	// Step 2: Wait for file list and select video files
	count, err := s.p.SelectVideoFiles(newID)
	if err != nil {
		// Clean up the new torrent if selection fails
		s.p.DeleteTorrent(newID)
		return fmt.Errorf("selecting video files: %w", err)
	}
	s.logger.Debug().Int("files", count).Msg("Selected video files")

	// Step 3: Delete the original dead torrent
	if err := s.p.DeleteTorrent(torrent.ID); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to delete original torrent")
		// Don't return error - the repair was successful
	}

	s.logger.Info().
		Str("oldId", torrent.ID).
		Str("newId", newID).
		Msg("Torrent repaired successfully")

	return nil
}

// RepairTorrents repairs multiple torrents
func (s *Service) RepairTorrents(torrents []*provider.Torrent, dryRun bool) (int, int) {
	if len(torrents) == 0 {
		return 0, 0
	}

	s.logger.Info().Int("count", len(torrents)).Msg("Starting torrent repairs")

	var succeeded, failed int
	for _, t := range torrents {
		if err := s.RepairTorrent(t, dryRun); err != nil {
			s.logger.Error().Err(err).Str("filename", t.Filename).Msg("Repair failed")
			failed++
		} else {
			succeeded++
		}
	}

	s.logger.Info().
		Int("succeeded", succeeded).
		Int("failed", failed).
		Msg("Torrent repairs completed")

	return succeeded, failed
}

// RepairTorrentByHash repairs a torrent using just its hash
func (s *Service) RepairTorrentByHash(hash string, dryRun bool) error {
	if dryRun {
		s.logger.Info().Str("hash", hash[:8]).Msg("[DRY-RUN] Would repair torrent by hash")
		return nil
	}

	// Add magnet
	newID, err := s.p.AddMagnet(hash)
	if err != nil {
		return fmt.Errorf("adding magnet: %w", err)
	}

	// Select video files
	count, err := s.p.SelectVideoFiles(newID)
	if err != nil {
		s.p.DeleteTorrent(newID)
		return fmt.Errorf("selecting video files: %w", err)
	}

	s.logger.Info().
		Str("hash", hash[:8]).
		Str("id", newID).
		Int("files", count).
		Msg("Torrent added by hash")

	return nil
}
