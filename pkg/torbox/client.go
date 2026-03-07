package torbox

import (
	"fmt"
	"sync"

	"github.com/robofuse/robofuse/internal/config"
	"github.com/robofuse/robofuse/internal/logger"
	"github.com/robofuse/robofuse/internal/request"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

// client.go configures the TorBox API client.

const (
	defaultHost = "https://api.torbox.app/v1/api"

	// TorBox default rate limits (more generous than Real-Debrid)
	defaultGeneralRateLimit  = 60 // req/min
	defaultTorrentsRateLimit = 60 // req/min
)

// Client is the TorBox API client.
type Client struct {
	Host   string
	APIKey string

	httpClient *request.Client

	logger zerolog.Logger
	config *config.Config

	mu sync.RWMutex
}

// New creates a new TorBox client.
func New(cfg *config.Config) *Client {
	log := logger.New("torbox")

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", cfg.Token),
	}

	generalRL := request.ParseRateLimitInt(cfg.GeneralRateLimit)
	if generalRL == nil {
		generalRL = rate.NewLimiter(rate.Limit(1.0), 1)
	}

	httpClient := request.New(
		request.WithHeaders(headers),
		request.WithRateLimiter(generalRL),
		request.WithLogger(log),
		request.WithMaxRetries(5),
		request.WithRetryableStatus(429, 502, 503),
	)

	return &Client{
		Host:       defaultHost,
		APIKey:     cfg.Token,
		httpClient: httpClient,
		logger:     log,
		config:     cfg,
	}
}
