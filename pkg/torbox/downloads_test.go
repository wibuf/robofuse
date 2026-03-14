package torbox

import "testing"

func TestIsBonusContent(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// Bonus content in subfolders
		{"Zootopia (2016)/Extras/Behind The Scenes.mkv", true},
		{"Movie/Bonus/Deleted Scene.mkv", true},
		{"Movie/Featurettes/Making Of.mkv", true},
		{"Movie/Special Features/Commentary.mkv", true},
		{"Movie/Behind The Scenes/Interview.mkv", true},
		{"Movie/Deleted Scenes/Alt Ending.mkv", true},
		{"Movie/Interviews/Director.mkv", true},
		{"Movie/Trailers/Trailer.mkv", true},
		// Main content (not bonus)
		{"Zootopia (2016)/Zootopia.2016.1080p.mkv", false},
		{"Movie.2024.mkv", false},
		{"Movie/Movie.mkv", false},
		// Edge cases
		{"extras.mkv", false}, // "extras" as filename, not folder
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isBonusContent(tt.path); got != tt.want {
				t.Errorf("isBonusContent(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsSampleFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"Sample.mkv", true},
		{"sample.mkv", true},
		{"SAMPLE.mkv", true},
		{"sample-video.mkv", true},
		{"sample.video.mkv", true},
		{"The.Movie.2024.mkv", false},
		{"Sampled.mkv", false},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if got := isSampleFile(tt.filename); got != tt.want {
				t.Errorf("isSampleFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}
