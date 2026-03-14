package organizer

import "testing"

func TestIsHexHash(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"a1a141eeeccce637f9fb2c82cad602954974", true},
		{"63f738535dbac7f06607155d89380fe7563", true},
		{"9480c0e831a93edecbe9cff4115d7a370", true},
		{"ABCDEF0123456789ABCDEF0123456789", true},
		{"abc123", false},          // too short
		{"The Dinosaurs", false},   // not hex
		{"Spider-Man", false},      // not hex
		{"One Piece S01E01", false}, // not hex
		{"", false},                // empty
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isHexHash(tt.input); got != tt.want {
				t.Errorf("isHexHash(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Release group removal
		{"The Dinosaurs .-MeGusta", "The Dinosaurs"},
		{"Show Name -SPARKS", "Show Name"},
		{"Some Title .-GroupName", "Some Title"},
		// ALL CAPS to Title Case
		{"ONE PIECE WHISKY BUSINE", "One Piece Whisky Busine"},
		{"NCIS", "Ncis"}, // short but still > 3 chars
		// Already clean titles pass through
		{"Tulsa King", "Tulsa King"},
		{"South Park", "South Park"},
		{"Yellowstone", "Yellowstone"},
		// Trailing junk removal
		{"Title Name .", "Title Name"},
		{"Title Name -", "Title Name"},
		{"Title Name _", "Title Name"},
		// Empty/hash pass through
		{"", ""},
		{"a1a141eeeccce637f9fb2c82cad602954974", "a1a141eeeccce637f9fb2c82cad602954974"},
		// Don't strip mid-title dashes (no space before dash)
		{"Spider-Man", "Spider-Man"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := cleanTitle(tt.input); got != tt.want {
				t.Errorf("cleanTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTitlesMatch(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		// Exact match
		{"Vladimir", "Vladimir", true},
		// Case insensitive
		{"vladimir", "Vladimir", true},
		// Plural handling
		{"Vladimir", "Vladimirs", true},
		{"vladimirs", "Vladimir", true},
		// With year suffix
		{"Tulsa King", "Tulsa King (2022)", true},
		// "the" prefix
		{"The Dinosaurs", "Dinosaurs", true},
		{"Dinosaurs", "The Dinosaurs", true},
		// "the" + plural
		{"The Dinosaur", "Dinosaurs", true},
		// Different titles should NOT match
		{"NCIS", "Scrubs", false},
		{"One Piece", "Naruto", false},
		// Don't false-match on trailing s for short words
		{"NCIS", "NCI", false},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			if got := titlesMatch(tt.a, tt.b); got != tt.want {
				t.Errorf("titlesMatch(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestStripGroup(t *testing.T) {
	tests := []struct {
		title, group, want string
	}{
		{"Research A True Life Adventure Grym", "Grym", "Research A True Life Adventure"},
		{"Zootopia Essa Cidade E O Bicho Grym", "Grym", "Zootopia Essa Cidade E O Bicho"},
		{"The Fall Guy", "", "The Fall Guy"},         // no group
		{"The Fall Guy", "SPARKS", "The Fall Guy"},   // group not in title
		{"Spider-Man", "Man", "Spider-Man"},          // group in title but not as suffix with space
		{"", "Grym", ""},                             // empty title
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			if got := stripGroup(tt.title, tt.group); got != tt.want {
				t.Errorf("stripGroup(%q, %q) = %q, want %q", tt.title, tt.group, got, tt.want)
			}
		})
	}
}

func TestNormalizeForMatch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Tulsa King (2022)", "tulsa king"},
		{"Vladimir", "vladimir"},
		{"  South Park  ", "south park"},
		{"NCIS", "ncis"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeForMatch(tt.input); got != tt.want {
				t.Errorf("normalizeForMatch(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
