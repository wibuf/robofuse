// Package organizer provides media file organization functionality.
// It parses torrent filenames using ptt-go and organizes them into
// Movies/Series/Anime folder structures.
package organizer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	ptt "github.com/itsrenoria/ptt-go"
	"github.com/rs/zerolog"
)

// organizer.go handles parsing and output path construction for media items.

// Result contains statistics from the organization process.
type Result struct {
	Processed int `json:"processed"`
	New       int `json:"new"`
	Deleted   int `json:"deleted"`
	Updated   int `json:"updated"`
	Skipped   int `json:"skipped"`
	Errors    int `json:"errors"`
}

// FileEntry represents a tracked file in the organizer database.
type FileEntry struct {
	DestPath    string `json:"dest_path"`
	RDID        string `json:"rd_id"`
	Type        string `json:"type"`
	DownloadURL string `json:"download_url,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// TrackingEntry represents an entry from the file tracking system.
type TrackingEntry struct {
	Link        string `json:"link"`
	DownloadURL string `json:"download_url,omitempty"`
	LastChecked string `json:"last_checked,omitempty"`
}

// Organizer handles media file organization.
type Organizer struct {
	baseDir      string
	libraryDir   string
	organizedDir string
	dbPath       string
	trackingPath string
	parser       *ptt.Parser
	logger       zerolog.Logger
	db           map[string]FileEntry
}

// Config holds organizer configuration.
type Config struct {
	BaseDir      string
	OrganizedDir string
	OutputDir    string
	TrackingFile string
	CacheDir     string
	Logger       zerolog.Logger
}

// New creates a new Organizer instance.
func New(cfg Config) *Organizer {
	parser := ptt.NewParser()
	ptt.AddDefaults(parser)

	libraryDir := cfg.OutputDir
	if libraryDir == "" {
		libraryDir = filepath.Join(cfg.BaseDir, "library")
	}

	organizedDir := cfg.OrganizedDir
	if organizedDir == "" {
		organizedDir = filepath.Join(cfg.BaseDir, "library-organized")
	}

	cacheDir := cfg.CacheDir
	if cacheDir == "" {
		cacheDir = filepath.Join(cfg.BaseDir, "cache")
	}

	trackingPath := cfg.TrackingFile
	if trackingPath == "" {
		trackingPath = filepath.Join(cfg.BaseDir, "cache", "file_tracking.json")
	}

	return &Organizer{
		baseDir:      cfg.BaseDir,
		libraryDir:   libraryDir,
		organizedDir: organizedDir,
		dbPath:       filepath.Join(cacheDir, "organizer_db.json"),
		trackingPath: trackingPath,
		parser:       parser,
		logger:       cfg.Logger,
		db:           make(map[string]FileEntry),
	}
}

// loadDB loads the organizer database from disk.
func (o *Organizer) loadDB() error {
	data, err := os.ReadFile(o.dbPath)
	if os.IsNotExist(err) {
		o.db = make(map[string]FileEntry)
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &o.db)
}

// saveDB saves the organizer database to disk.
func (o *Organizer) saveDB() error {
	if err := os.MkdirAll(filepath.Dir(o.dbPath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(o.db, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(o.dbPath, data, 0644)
}

// loadTracking loads the file tracking database.
func (o *Organizer) loadTracking() (map[string]TrackingEntry, error) {
	data, err := os.ReadFile(o.trackingPath)
	if os.IsNotExist(err) {
		return make(map[string]TrackingEntry), nil
	}
	if err != nil {
		return nil, err
	}

	var tracking map[string]TrackingEntry
	if err := json.Unmarshal(data, &tracking); err != nil {
		return nil, err
	}
	return tracking, nil
}

var rdIDRegex = regexp.MustCompile(`/d/([a-zA-Z0-9]+)`)

// getRDIDFromLink extracts the Real-Debrid ID from a link.
func getRDIDFromLink(link string) string {
	if link == "" {
		return ""
	}
	match := rdIDRegex.FindStringSubmatch(link)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

var illegalCharsRegex = regexp.MustCompile(`[<>:"/\\|?*]`)

// cleanFilename removes illegal filesystem characters.
func cleanFilename(name string) string {
	return illegalCharsRegex.ReplaceAllString(name, "")
}

// isHexHash returns true if s looks like a hex hash (16+ contiguous hex chars, no spaces).
func isHexHash(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 16 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// trailingGroupRegex matches release group tags that ptt-go missed.
// Matches " -GroupName" and " .-GroupName" at end of title.
// Won't match mid-title dashes like "Spider-Man" (no space before dash).
var trailingGroupRegex = regexp.MustCompile(`\s+\.?-[A-Za-z0-9]+$`)

// cleanTitle post-processes a ptt-go parsed title to remove leftover junk.
func cleanTitle(title string) string {
	if title == "" {
		return title
	}

	// Skip cleaning if the title is a hash — caller should handle this
	if isHexHash(title) {
		return title
	}

	// Strip trailing release group patterns (e.g., " .-MeGusta", " -SPARKS")
	title = trailingGroupRegex.ReplaceAllString(title, "")

	// Strip trailing dots, dashes, underscores, spaces
	title = strings.TrimRight(title, " .-_")

	// Convert ALL CAPS to Title Case (e.g., "ONE PIECE" → "One Piece")
	if len(title) > 3 && title == strings.ToUpper(title) {
		title = toTitleCase(title)
	}

	return strings.TrimSpace(title)
}

// stripGroup removes the release group name from the end of a title
// if ptt-go extracted it but left it in the title string.
// e.g., title="Research A True Life Adventure Grym", group="Grym" → "Research A True Life Adventure"
func stripGroup(title, group string) string {
	if group == "" || title == "" {
		return title
	}
	if strings.HasSuffix(strings.ToLower(title), " "+strings.ToLower(group)) {
		title = strings.TrimSpace(title[:len(title)-len(group)-1])
	}
	return title
}

// toTitleCase converts "ONE PIECE" to "One Piece".
func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) == 0 {
			continue
		}
		runes := []rune(strings.ToLower(w))
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

// normalizeForMatch strips year suffixes and normalizes a title for fuzzy comparison.
func normalizeForMatch(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// Strip year suffix like " (2022)"
	if idx := strings.LastIndex(s, " ("); idx >= 0 {
		rest := s[idx:]
		if len(rest) == 7 && rest[len(rest)-1] == ')' {
			s = s[:idx]
		}
	}
	return s
}

// titlesMatch returns true if two titles refer to the same show,
// accounting for pluralization and "the" prefix differences.
func titlesMatch(a, b string) bool {
	a = normalizeForMatch(a)
	b = normalizeForMatch(b)

	if a == b {
		return true
	}

	// Check plural: "vladimir" vs "vladimirs"
	// Only for multi-word titles or words with 5+ chars to avoid
	// false positives on acronyms like NCIS/NCI.
	if pluralMatch(a, b) {
		return true
	}

	// Check with/without "the " prefix
	aNoThe := strings.TrimPrefix(a, "the ")
	bNoThe := strings.TrimPrefix(b, "the ")
	if aNoThe == bNoThe {
		return true
	}
	if pluralMatch(aNoThe, bNoThe) {
		return true
	}

	return false
}

// pluralMatch returns true if a+"s"==b or b+"s"==a,
// but only when the shorter form is multi-word or 5+ chars
// (avoids false positives on acronyms like NCIS vs NCI).
func pluralMatch(a, b string) bool {
	if a+"s" == b && (len(a) >= 5 || strings.Contains(a, " ")) {
		return true
	}
	if b+"s" == a && (len(b) >= 5 || strings.Contains(b, " ")) {
		return true
	}
	return false
}

// findExistingSeriesFolder checks if a folder for the series already exists.
// Uses fuzzy matching to handle pluralization ("Vladimir" vs "Vladimirs"),
// "the" prefix differences, and year suffix variations.
func (o *Organizer) findExistingSeriesFolder(baseFolder, title string, year int) string {
	searchDir := filepath.Join(o.organizedDir, baseFolder)
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return ""
	}

	targetWithYear := title
	if year > 0 {
		targetWithYear = fmt.Sprintf("%s (%d)", title, year)
	}

	// Pass 1: Exact match (case-insensitive)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.EqualFold(entry.Name(), targetWithYear) {
			return entry.Name()
		}
	}

	// Pass 2: Fuzzy match using titlesMatch (handles plurals, "the" prefix)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if titlesMatch(title, entry.Name()) {
			return entry.Name()
		}
	}

	return ""
}

// getContentTypeAndPath determines content type and destination path.
func (o *Organizer) getContentTypeAndPath(parsed, parentParsed *ptt.TorrentInfo, filename, rdID string) (string, string) {
	// Extract info from filename
	fTitle := stripGroup(cleanTitle(parsed.Title), parsed.Group)
	fYear := parsed.Year
	fSeason := parsed.Seasons
	fEpisode := parsed.Episodes
	fAnime := parsed.Anime

	// Extract info from parent — skip if parent title is a hex hash
	// (TorBox sometimes returns the info hash as the torrent name)
	pTitle := ""
	pYear := 0
	var pSeason, pEpisode []int
	pAnime := false
	if parentParsed != nil && !isHexHash(parentParsed.Title) {
		pGroup := ""
		if parentParsed != nil {
			pGroup = parentParsed.Group
		}
		pTitle = stripGroup(cleanTitle(parentParsed.Title), pGroup)
		pYear = parentParsed.Year
		pSeason = parentParsed.Seasons
		pEpisode = parentParsed.Episodes
		pAnime = parentParsed.Anime
	}

	// Determine if series
	isSeriesFilename := len(fSeason) > 0 || len(fEpisode) > 0 || fAnime
	isSeriesParent := len(pSeason) > 0 || len(pEpisode) > 0 || pAnime

	var finalType, title string
	var year int
	var season, episode []int

	if isSeriesParent {
		if pAnime {
			finalType = "anime"
		} else {
			finalType = "series"
		}

		if pTitle != "" {
			title = pTitle
		} else {
			title = "Unknown"
		}

		if pYear > 0 {
			year = pYear
		} else {
			year = fYear
		}

		if len(fSeason) > 0 {
			season = fSeason
		} else {
			season = pSeason
		}

		if len(fEpisode) > 0 {
			episode = fEpisode
		}
	} else if isSeriesFilename {
		if fAnime {
			finalType = "anime"
		} else {
			finalType = "series"
		}
		if fTitle != "" {
			title = fTitle
		} else {
			title = "Unknown"
		}
		year = fYear
		season = fSeason
		episode = fEpisode
	} else {
		finalType = "movie"
		if fTitle != "" {
			title = fTitle
		} else if pTitle != "" {
			title = pTitle
		} else {
			title = "Unknown"
		}
		if fYear > 0 {
			year = fYear
		} else {
			year = pYear
		}
	}

	// Determine base folder
	var baseFolder string
	switch finalType {
	case "anime":
		baseFolder = "Anime"
	case "series":
		baseFolder = "Series"
	default:
		baseFolder = "Movies"
	}

	// Check for existing folder
	existingFolder := o.findExistingSeriesFolder(baseFolder, title, year)
	var formattedTitle string
	if existingFolder != "" {
		formattedTitle = existingFolder
	} else {
		formattedTitle = title
		if year > 0 {
			formattedTitle = fmt.Sprintf("%s (%d)", formattedTitle, year)
		}
		formattedTitle = cleanFilename(formattedTitle)
	}

	// ID suffix
	idSuffix := ""
	if rdID != "" {
		idSuffix = fmt.Sprintf(" [%s]", rdID)
	}

	// Extension
	ext := filepath.Ext(filename)

	var destPath string
	if finalType == "movie" {
		finalFilename := cleanFilename(fmt.Sprintf("%s%s%s", formattedTitle, idSuffix, ext))
		destPath = filepath.Join("Movies", formattedTitle, finalFilename)
	} else {
		// Series or Anime
		var seasonFolder string
		if len(season) > 0 {
			seasonFolder = fmt.Sprintf("Season %02d", season[0])
		} else {
			seasonFolder = "Season Unknown"
		}

		var finalFilename string
		if len(episode) > 0 {
			var epStr string
			if len(season) > 0 {
				epStr = fmt.Sprintf("S%02dE%02d", season[0], episode[0])
			} else {
				epStr = fmt.Sprintf("E%02d", episode[0])
			}
			finalFilename = cleanFilename(fmt.Sprintf("%s %s%s%s", title, epStr, idSuffix, ext))
		} else {
			partName := fTitle
			if partName == "" {
				partName = "Unknown"
			}
			if strings.EqualFold(partName, title) {
				finalFilename = cleanFilename(fmt.Sprintf("%s%s%s", title, idSuffix, ext))
			} else {
				finalFilename = cleanFilename(fmt.Sprintf("%s - %s%s%s", title, partName, idSuffix, ext))
			}
		}

		destPath = filepath.Join(baseFolder, formattedTitle, seasonFolder, finalFilename)
	}

	return finalType, destPath
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// Run executes the organization process.
func (o *Organizer) Run() Result {
	result := Result{}

	if err := o.loadDB(); err != nil {
		o.logger.Error().Err(err).Msg("Failed to load organizer database")
		return result
	}

	tracking, err := o.loadTracking()
	if err != nil {
		o.logger.Error().Err(err).Msg("Failed to load tracking database")
		return result
	}

	result.Processed = len(tracking)
	currentSourcePaths := make(map[string]bool)
	newState := make(map[string]FileEntry)

	for relPath, meta := range tracking {
		sourceFullPath := filepath.Join(o.libraryDir, relPath)
		if !fileExists(sourceFullPath) {
			continue
		}
		currentSourcePaths[relPath] = true

		// Check if already organized and up to date
		if prevEntry, exists := o.db[relPath]; exists {
			currentID := getRDIDFromLink(meta.Link)
			destFullPath := filepath.Join(o.organizedDir, prevEntry.DestPath)
			sameURL := meta.DownloadURL != "" && prevEntry.DownloadURL == meta.DownloadURL
			if prevEntry.RDID == currentID && fileExists(destFullPath) && (sameURL || meta.DownloadURL == "") {
				newState[relPath] = prevEntry
				result.Skipped++
				continue
			}
		}

		// Needs organization

		// Parse filename
		filename := filepath.Base(relPath)
		nameNoExt := strings.TrimSuffix(filename, filepath.Ext(filename))
		parsed := o.parser.Parse(nameNoExt)

		// Parse parent folder
		parentRelDir := filepath.Dir(relPath)
		parentFolderName := ""
		if parentRelDir != "" && parentRelDir != "." {
			parentFolderName = filepath.Base(parentRelDir)
		}

		var parentParsed *ptt.TorrentInfo
		if parentFolderName != "" && !isHexHash(parentFolderName) {
			parentParsed = o.parser.Parse(parentFolderName)
		}

		rdID := getRDIDFromLink(meta.Link)

		// Determine destination
		contentType, destRelPath := o.getContentTypeAndPath(parsed, parentParsed, filename, rdID)
		destFullPath := filepath.Join(o.organizedDir, destRelPath)

		// Copy file
		if err := copyFile(sourceFullPath, destFullPath); err != nil {
			o.logger.Error().Err(err).Str("path", relPath).Msg("Failed to organize file")
			result.Errors++
			continue
		}

		newState[relPath] = FileEntry{
			DestPath:    destRelPath,
			RDID:        rdID,
			Type:        contentType,
			DownloadURL: meta.DownloadURL,
			UpdatedAt:   meta.LastChecked,
		}
		result.New++
	}

	// Cleanup deleted files
	for oldSrcPath, oldEntry := range o.db {
		if !currentSourcePaths[oldSrcPath] {
			destFull := filepath.Join(o.organizedDir, oldEntry.DestPath)
			if fileExists(destFull) {
				if err := os.Remove(destFull); err == nil {
					result.Deleted++
					// Try to remove empty parent directories
					o.cleanEmptyDirs(filepath.Dir(destFull))
				}
			}
		}
	}

	// Save new state
	o.db = newState
	if err := o.saveDB(); err != nil {
		o.logger.Error().Err(err).Msg("Failed to save organizer database")
	}

	return result
}

// cleanEmptyDirs removes empty directories up to the organized root.
func (o *Organizer) cleanEmptyDirs(dir string) {
	for dir != o.organizedDir && strings.HasPrefix(dir, o.organizedDir) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
