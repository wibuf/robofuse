## Feature Request: Add TorBox as a Debrid Provider

### Summary

Add [TorBox](https://torbox.app) as an alternative debrid provider alongside the existing Real-Debrid support. This would allow users to choose their preferred debrid service while sharing the same STRM generation, file organization, and watch mode infrastructure.

### Motivation

RoboFuse currently only supports Real-Debrid. TorBox is a growing debrid service with a well-documented REST API ([api.torbox.app/docs](https://api.torbox.app/docs)) that supports torrents, web downloads, and usenet. Adding provider-agnostic support benefits the broader community and makes RoboFuse more versatile.

### Proposed Implementation

This feature involves three main areas of work:

#### 1. Introduce a Provider Interface

Create a common `Provider` interface in `pkg/provider/` that abstracts the debrid service operations:

- `GetTorrents()` — fetch downloaded and dead/errored torrents
- `GetDownloads()` — fetch completed downloads with streaming URLs
- `GetDownloadURL()` — obtain a direct download link for a specific file
- `AddMagnet()` — add a torrent via magnet hash
- `DeleteTorrent()` — remove a torrent
- `CheckLink()` — validate a link is still active

Both the existing Real-Debrid client and the new TorBox client would implement this interface.

#### 2. Implement TorBox Client (`pkg/torbox/`)

New package mirroring the structure of `pkg/realdebrid/`:

- **`client.go`** — HTTP client setup, rate limiter, Bearer token auth
- **`torrents.go`** — `POST /v1/api/torrents/createtorrent`, `GET /v1/api/torrents/mylist`, `POST /v1/api/torrents/controltorrent`
- **`downloads.go`** — `GET /v1/api/torrents/requestdl` (replaces RD's unrestrict concept)
- **`types.go`** — TorBox API response models

**Key API differences from Real-Debrid:**

| Concept | Real-Debrid | TorBox |
|---|---|---|
| Base URL | `api.real-debrid.com/rest/1.0` | `api.torbox.app/v1/api` |
| Auth | `Authorization: Bearer <token>` | `Authorization: Bearer <token>` |
| Add torrent | `POST /torrents/addMagnet` | `POST /torrents/createtorrent` |
| List torrents | `GET /torrents` | `GET /torrents/mylist` |
| Delete torrent | `DELETE /torrents/delete/{id}` | `POST /torrents/controltorrent` (op=delete) |
| Get download URL | `POST /unrestrict/link` (per-link) | `GET /torrents/requestdl?torrent_id=X&file_id=Y` (per-file) |
| Link lifetime | ~7 days | ~3 hours |

**Important:** TorBox returns download URLs directly per file within a torrent (no separate "unrestrict" step), and links are shorter-lived (~3 hours vs ~7 days). The STRM refresh/expiry logic needs to account for this.

#### 3. Refactor Sync Layer for Provider Abstraction

- Replace `rd *realdebrid.Client` in `pkg/sync/sync.go` with a `provider.Provider` interface
- Abstract the sync flow so it doesn't assume RD's unrestrict model
- The core loop (fetch torrents → find missing → get URLs → write STRMs) remains the same
- Provider-specific rate limit defaults

#### 4. Configuration Changes

Add a `provider` field to `config.json`:

```json
{
  "provider": "torbox",
  "token": "TORBOX_API_KEY"
}
```

Default to `"real-debrid"` for backwards compatibility.

### API Reference

- TorBox OpenAPI spec: https://api.torbox.app/openapi.json
- TorBox Swagger docs: https://api.torbox.app/docs
- TorBox Postman collection: https://www.postman.com/torbox/torbox/overview

### Checklist

- [ ] Define `Provider` interface in `pkg/provider/`
- [ ] Implement TorBox client in `pkg/torbox/`
- [ ] Refactor `pkg/realdebrid/` to implement `Provider` interface
- [ ] Refactor `pkg/sync/` to use `Provider` interface
- [ ] Update config to support provider selection
- [ ] Handle TorBox's shorter link expiry in STRM refresh logic
- [ ] Update README with TorBox configuration instructions
- [ ] Test with real TorBox account
