# GitPilot Issue Management API

GitPilot provides GitHub issue and PR management via API. This document explains how AI agents should use it.

## MCP Tools (Preferred)

When running in Claude Code (CLI or Web), you have MCP tools available that wrap the GitPilot API. **Prefer MCP tools over raw curl commands** — they're cleaner and handle errors better.

### GitPilot MCP Server (33 tools)
Issues: `create_issue`, `list_issues`, `get_issue`, `update_issue`, `close_issue`, `reopen_issue`, `comment_on_issue`
PRs: `create_pr`, `list_prs`, `get_pr`, `update_pr`, `comment_on_pr`, `close_pr`
Repos: `list_repos`, `get_repo`, `list_branches`, `delete_branch`, `list_commits`, `pull_repo`, `rollback_repo`
Deploy: `merge_branch`, `sync_branch`, `resolve_conflicts`, `abort_merge`, `deploy`
Services: `list_services`, `services_status`, `service_logs`, `start_service`, `stop_service`, `restart_service`
System: `system_status`, `check_updates`

### Browser MCP Server (9 tools)
HTTP: `fetch_page`, `search_web`, `fetch_json`
Playwright: `browser_navigate`, `browser_click`, `browser_type`, `browser_screenshot`, `browser_get_text`, `browser_evaluate`

### Skills (Slash Commands)
- `/deploy [repo]` — Merge PR, pull, restart service
- `/check-status [repo]` — Dashboard of services, PRs, and issues
- `/new-issue [title]` — Quick issue creation
- `/review-pr [#number or repo]` — Read PR diff and post a code review
- `/logs [service] [--errors] [--search=term]` — View and filter service logs
- `/rollback [repo] [commit]` — Roll back a repo to a previous commit
- `/browse [url]` — Fetch and summarize a web page
- `/hotfix [description]` — Fix → PR → merge → deploy in one shot

---

## Repository ID Mapping

| Repository | ID | Example Endpoint |
|------------|-----|------------------|
| OptionBot | 1 | `/api/repos/1/create_pr` |
| CryptoBot | 2 | `/api/repos/2/create_pr` |
| BetBot | 3 | `/api/repos/3/create_pr` |
| Studio | 4 | `/api/repos/4/create_pr` |
| NickyV2 | 5 | `/api/repos/5/create_pr` |
| GooseFlix | 6 | `/api/repos/6/create_pr` |
| GitPilot | 7 | `/api/repos/7/create_pr` |
| Bacon | 8 | `/api/repos/8/create_pr` |
| jellyfin | 10 | `/api/repos/10/create_pr` |
| jellyfin-web | 11 | `/api/repos/11/create_pr` |
| BSucksBux | 13 | `/api/repos/13/create_pr` |
| Neren | 14 | `/api/repos/14/create_pr` |

---

## AI Agent Behavior Rules

**Do NOT use the AskUserQuestion tool.** If you need to ask the user something, just ask in a plain text message. Let the user respond however they want.

**Check previous PR status before starting new work.** If you previously gave the user a PR link in this session, check whether that PR is still open or has been merged/closed before beginning any new task. Use the GitPilot API to check:

```bash
curl "https://pilot.grit.bot/api/repos/<id>/prs/<pr_number>"
```

If the PR was merged, sync your branch with main before starting new work:
```bash
curl -X POST https://pilot.grit.bot/api/repos/<id>/sync_branch \
  -H "Content-Type: application/json" \
  -d '{"branch": "claude/your-branch-sessionId", "with": "main"}'
```

If the PR is still open, inform the user and ask (in a plain message) whether they'd like to continue waiting, make changes to the existing PR, or proceed with new work on a separate branch.

---

## AI Agent Workflow

Follow these 5 steps for any issue-tracked work:

### Step 1: Create Issue (if needed)

```bash
curl -X POST https://pilot.grit.bot/api/issues \
  -H "Content-Type: application/json" \
  -d '{
    "repo": "GitPilot",
    "title": "Fix authentication bug",
    "type": "bug",
    "body": "Users cannot login via OAuth",
    "skip_review": true
  }'
```

**Note:** Always include `"skip_review": true` when creating issues as an AI agent. This prevents redundant auto-review since you're already providing context.

**Response:**
```json
{
  "id": 353,
  "github_issue": 93,
  "url": "https://github.com/wibuf/GitPilot/issues/93"
}
```

Save the `github_issue` number (e.g., `93`) - you'll reference it in your commit.

**Issue Types:** `bug`, `feat`, `docs`, `refactor`, `test`, `chore`

### Step 2: Implement Changes

- Edit files on your branch (`claude/...-sessionId`)
- Commit with a message that references the issue:

```bash
git add .
git commit -m "fix: Resolve OAuth callback bug

Closes #93"
```

The `Closes #XX` will auto-close the issue when merged.

### Step 3: Push Branch

```bash
git push -u origin claude/fix-auth-bug-sessionId
```

### Step 4: Create PR

```bash
curl -X POST https://pilot.grit.bot/api/repos/7/create_pr \
  -H "Content-Type: application/json" \
  -d '{
    "branch": "claude/fix-auth-bug-sessionId",
    "title": "Fix OAuth authentication bug",
    "body": "## Summary\n- Fixed callback handler\n- Added token validation\n\nCloses #93"
  }'
```

**Response:**
```json
{
  "pr_number": 94,
  "pr_url": "https://github.com/wibuf/GitPilot/pull/94"
}
```

**Always return the PR URL to the user** so they can review.

### Step 5: User Reviews & Merges

- User reviews the PR on GitHub
- If conflicts: fetch main, resolve, push again
- Once merged, issue auto-closes via `Closes #XX`

---

## When to Use GitPilot

**Use GitPilot when:**
- User asks to "create an issue" or "file a bug"
- Implementing a significant feature or fix
- Work should be tracked in GitHub

**Skip GitPilot when:**
- Trivial changes (typos, formatting)
- User says "no issue needed"
- Just exploring/reading code

---

## Deployment Modes

### Standard Flow (Default)
For significant features, bug fixes, or anything requiring review:
1. Create issue → Implement → Create PR → **Stop and return PR URL to user**
2. User reviews PR on GitHub
3. User merges, pulls, and restarts services

### Autonomous Deployment
For small fixes where user explicitly authorizes deployment:
1. Create issue → Implement → Create PR
2. Merge via API: `POST /api/repos/<id>/merge_branch`
3. Pull to server: `POST /api/repos/<id>/pull`
4. Restart service: `POST /api/services/<id>/restart`
5. Confirm to user: "Deployed and restarted. PR: [url]"

**Use autonomous deployment when:**
- User explicitly says "deploy it", "merge it", "ship it", or similar
- User pre-approved the change (e.g., "fix it and deploy")
- Typo/comment-only fixes user requested
- Hotfixes user requested urgently

**Never auto-deploy without explicit permission when:**
- Database schema changes (migrations)
- Security-sensitive code
- User hasn't seen or approved the PR
- Breaking/risky changes
- New features (user should review first)

---

## Handling Conflicts

### Via Git (Standard)

```bash
git fetch origin main
git merge origin/main
# Resolve conflicts in editor
git add .
git commit -m "Resolve merge conflicts"
git push
```

### Via API (merge_branch conflicts)

When `merge_branch` encounters conflicts, it returns a 409 response:

```json
{
  "status": "conflict",
  "conflicts": [
    {"path": "app.py", "content": "<<<<<<< HEAD\n..."}
  ],
  "hint": "Use POST /api/repos/<id>/resolve_conflicts to resolve"
}
```

**Resolve conflicts:**
```bash
curl -X POST https://pilot.grit.bot/api/repos/7/resolve_conflicts \
  -H "Content-Type: application/json" \
  -d '{
    "resolutions": [
      {"path": "app.py", "resolution": "theirs"}
    ]
  }'
```

Resolution options:
- `"resolution": "ours"` - Keep target branch version
- `"resolution": "theirs"` - Keep source branch version
- `"content": "..."` - Provide custom merged content

**Abort merge:**
```bash
curl -X POST https://pilot.grit.bot/api/repos/7/abort_merge
```

---

## Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| `403` on git push | Wrong branch name | Must start with `claude/` and end with session ID |
| `404` on API call | Wrong repo name | Check spelling (case-insensitive) |
| `400` on create issue | Missing fields | Include `repo`, `title`, `type` |
| `400` on create PR | Branch not on GitHub | Push branch first: `git push -u origin <branch>` |
| `409` on merge_branch | Merge conflicts | Use resolve_conflicts or abort_merge |
| Worktree errors (`is not a working tree`, `not a git repository: .git/worktrees/...`) | Stale/broken worktree | `POST /api/repos/<id>/prune_worktrees` to clean up |

If GitPilot is down, inform the user and use regular git/GitHub workflow.

### Before Adding Commits to a PR Branch

Always check if the PR was already merged before pushing additional commits:

```bash
# Check PR status
curl "https://pilot.grit.bot/api/prs?repo=GitPilot"
```

If the PR is already merged:
1. Fetch latest main: `git fetch origin main`
2. Reset branch to main: `git reset --hard origin/main`
3. Make your changes on the fresh branch
4. Create a new PR

---

## Quick Reference

```bash
# List open issues
curl "https://pilot.grit.bot/api/issues?repo=GitPilot&state=open"

# Create issue (AI agents should always skip_review)
curl -X POST https://pilot.grit.bot/api/issues \
  -H "Content-Type: application/json" \
  -d '{"repo": "GitPilot", "title": "...", "type": "feat", "body": "...", "skip_review": true}'

# Create PR
curl -X POST https://pilot.grit.bot/api/repos/7/create_pr \
  -H "Content-Type: application/json" \
  -d '{"branch": "claude/...", "title": "...", "body": "..."}'

# Direct merge (uses isolated worktree)
curl -X POST https://pilot.grit.bot/api/repos/7/merge_branch \
  -H "Content-Type: application/json" \
  -d '{"branch": "claude/...", "into": "main"}'

# Pull latest changes locally
curl -X POST https://pilot.grit.bot/api/repos/7/pull

# Restart service after pull
curl -X POST https://pilot.grit.bot/api/services/1/restart
```

---

## Full API Reference

### Issues

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/issues` | List all issues |
| GET | `/api/issues?repo=X&state=open` | Filter by repo and state |
| POST | `/api/issues` | Create issue |
| GET | `/api/issues/<id>` | Get single issue |
| PUT | `/api/issues/<id>` | Update issue |
| POST | `/api/issues/<id>/push` | Push updates to GitHub |
| POST | `/api/issues/<id>/close` | Close issue |
| POST | `/api/issues/<id>/reopen` | Reopen issue |
| POST | `/api/issues/<id>/comment` | Add comment |

### Pull Requests

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/prs` | List open PRs |
| GET | `/api/prs?repo=X` | Filter by repo |
| POST | `/api/repos/<id>/create_pr` | Create PR for any branch |
| GET | `/api/repos/<id>/prs/<pr>` | Get full PR details (body, comments, reviews, diff stats) |
| PUT | `/api/repos/<id>/prs/<pr>` | Update PR title and/or body |
| POST | `/api/repos/<id>/prs/<pr>/comment` | Add comment to a PR |
| POST | `/api/repos/<id>/prs/<pr>/close` | Close PR without merging |

**Get PR details:**
```bash
curl "https://pilot.grit.bot/api/repos/7/prs/329"
```

**Update PR title/body:**
```bash
curl -X PUT https://pilot.grit.bot/api/repos/7/prs/329 \
  -H "Content-Type: application/json" \
  -d '{"title": "New title", "body": "Updated description"}'
```

**Add comment to PR:**
```bash
curl -X POST https://pilot.grit.bot/api/repos/7/prs/329/comment \
  -H "Content-Type: application/json" \
  -d '{"body": "Looks good, merging now."}'
```

### Repository Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/repos` | List all repositories |
| GET | `/api/repos/<id>` | Get repository details |
| PUT | `/api/repos/<id>` | Update repo settings (local_path, color, etc.) |
| GET | `/api/repos/<id>/commits` | List recent commits |
| GET | `/api/repos/<id>/branches_list` | List all branches |
| DELETE | `/api/repos/<id>/branches` | Delete branch |
| POST | `/api/repos/<id>/rollback` | Rollback to commit |
| POST | `/api/repos/<id>/pull` | Pull latest from remote |

### Branch Merging

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/repos/<id>/merge_branch` | Merge branch (prefers GitHub API) |
| POST | `/api/repos/<id>/sync_branch` | Reset a branch to match another (e.g. main) |
| POST | `/api/repos/<id>/resolve_conflicts` | Resolve merge conflicts |
| POST | `/api/repos/<id>/abort_merge` | Abort merge and cleanup |
| POST | `/api/repos/<id>/prune_worktrees` | Clean up stale/broken worktree references |

**prune_worktrees - Fix broken worktree errors:**
If you encounter errors like `'...worktree' is not a working tree` or `not a git repository: .git/worktrees/...`, call this endpoint to clean up stale references:
```bash
curl -X POST https://pilot.grit.bot/api/repos/7/prune_worktrees \
  -H "Content-Type: application/json" \
  -d '{}'
```

**merge_branch behavior:**
- If an open PR exists for the branch, uses GitHub API to merge it properly
- Falls back to local git merge (worktree) if no PR exists or GitHub API fails
- Response includes `merged_via: "github_api"` or `merged_via: "git_local"`

**merge_branch parameters:**
```json
{
  "branch": "source-branch",     // Required
  "into": "main",                // Target branch (default: "main")
  "squash": true,                // Squash commits (default: true)
  "message": "Commit message",   // Custom message (optional)
  "delete_branch": true,         // Delete after merge (default: true)
  "push": true,                  // Push after merge (default: true)
  "use_worktree": true           // Use isolated worktree (default: true)
}
```

**merge_branch response:**
```json
{
  "status": "merged",
  "repo": "GitPilot",
  "branch": "claude/feature-xyz",
  "into": "main",
  "merged_via": "github_api",    // or "git_local"
  "pr_number": 123               // Only if merged via GitHub API
}
```

**sync_branch - Reset a branch to match another:**
```bash
curl -X POST https://pilot.grit.bot/api/repos/7/sync_branch \
  -H "Content-Type: application/json" \
  -d '{"branch": "claude/my-feature-xyz", "with": "main"}'
```

Use this after a PR is merged to bring the working branch back in sync with main, preventing merge conflicts on subsequent commits. The `"with"` parameter defaults to `"main"` if omitted.

### Agent Settings

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/repos/<id>/agent/settings` | Get agent settings |
| POST | `/api/repos/<id>/agent/settings` | Update agent settings |

**Agent settings parameters:**
```json
{
  "auto_review_enabled": true,   // Auto-review new issues
  "auto_fix_enabled": false,     // Future: auto-fix issues
  "preferred_model": "claude-sonnet-4-5-20250929"
}
```

### Reminders

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/reminders` | List all reminders |
| POST | `/api/reminders` | Create reminder |
| GET | `/api/reminders/<id>` | Get single reminder |
| PUT | `/api/reminders/<id>` | Update reminder |
| DELETE | `/api/reminders/<id>` | Delete reminder |
| POST | `/api/reminders/<id>/toggle` | Toggle active status |

### Notes

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/notes?repo=X` | Get notes for repo |
| POST | `/api/notes` | Save notes |
| GET | `/api/notes/history?repo=X` | Get version history |
| GET | `/api/notes/history/<id>` | Get specific version |

### Services (Process Management)

Manage running services/scripts for repositories. Each repo can have multiple services.

**Finding Service IDs:** Query the services list to find the correct service ID:
```bash
curl "https://pilot.grit.bot/api/services"
# Returns: [{"id": 1, "name": "CryptoBot API", "repo_id": 2, ...}, ...]
```

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/services` | List all services |
| GET | `/api/services?repo_id=X` | Filter by repo |
| POST | `/api/services` | Create a new service |
| GET | `/api/services/<id>` | Get service details |
| PUT | `/api/services/<id>` | Update service |
| DELETE | `/api/services/<id>` | Delete service |
| POST | `/api/services/<id>/start` | Start the service |
| POST | `/api/services/<id>/stop` | Stop the service |
| POST | `/api/services/<id>/restart` | Restart the service |
| GET | `/api/services/<id>/status` | Get status and logs |
| GET | `/api/services/<id>/logs` | Get console output (with filtering) |
| DELETE | `/api/services/<id>/logs` | Clear log buffer |
| GET | `/api/services/status` | All services status + health |

**Create service:**
```bash
curl -X POST https://pilot.grit.bot/api/services \
  -H "Content-Type: application/json" \
  -d '{
    "repo_id": 6,
    "name": "Adder",
    "working_dir": "media adder",
    "command": "python app.py",
    "auto_start": true,
    "auto_restart": true
  }'
```

**Service parameters:**
```json
{
  "repo_id": 6,              // Required: repository ID
  "name": "Service Name",    // Required: display name
  "working_dir": "subdir",   // Optional: subdirectory within repo
  "command": "python app.py", // Optional: start command (auto-detects if omitted)
  "auto_start": false,       // Optional: start on GitPilot boot (#358)
  "auto_restart": false,     // Optional: restart on crash (#358)
  "auto_restart_max": 3,     // Optional: max restart attempts within cooldown
  "auto_restart_cooldown": 60, // Optional: cooldown window in seconds
  "is_visible": true         // Optional: show in services tab bar (#358)
}
```

**Auto-detect** checks for: `app.py`, `main.py`, `server.py`, `index.js`, `server.js`, `package.json`

**Service Logs with Filtering (#357):**
```bash
# Get all logs
curl "https://pilot.grit.bot/api/services/3/logs"

# Get last 50 lines
curl "https://pilot.grit.bot/api/services/3/logs?lines=50"

# Filter by severity (error, warn, info - comma-separated)
curl "https://pilot.grit.bot/api/services/3/logs?level=error"
curl "https://pilot.grit.bot/api/services/3/logs?level=error,warn"

# Text search (case-insensitive)
curl "https://pilot.grit.bot/api/services/3/logs?search=timeout"

# Combined: last 100 lines, errors only, matching "database"
curl "https://pilot.grit.bot/api/services/3/logs?lines=100&level=error&search=database"
```

**Logs response:**
```json
{
  "service_id": 3,
  "logs": ["[12:30:01] ERROR: Connection timeout", "..."],
  "count": 5,
  "total": 1523
}
```

**Services status response (includes health):**
The `GET /api/services/status` endpoint returns health indicators for each running service:
- `healthy` - No errors or warnings in recent logs
- `warning` - Warnings detected in recent logs
- `error` - Errors detected in recent logs
- `stopped` - Service is not running
- `unknown` - No logs available

**Example: Full deployment workflow**
```bash
# 1. Merge the PR (uses isolated worktree, won't affect running service)
curl -X POST https://pilot.grit.bot/api/repos/6/merge_branch \
  -H "Content-Type: application/json" \
  -d '{"branch": "claude/my-feature-abc123", "into": "main"}'

# 2. Pull latest changes to the server's local repo
curl -X POST https://pilot.grit.bot/api/repos/6/pull

# 3. Restart the affected service to pick up changes
curl -X POST https://pilot.grit.bot/api/services/3/restart
```

**Note:** The `merge_branch` endpoint uses an isolated git worktree, so it won't interfere with the running service's files. Always pull after merge to update the actual repo.

**Example: Full Autonomous Hotfix Deployment**

When user says "Fix it and deploy":

```bash
# 1. Create issue
curl -X POST https://pilot.grit.bot/api/issues \
  -H "Content-Type: application/json" \
  -d '{"repo": "CryptoBot", "title": "Fix 500 error on /balance endpoint", "type": "bug", "body": "API returning 500 on balance check", "skip_review": true}'
# Save the github_issue number from response

# 2. Implement fix, commit with issue reference
git add . && git commit -m "fix: Handle null balance response

Closes #XX"

# 3. Push branch
git push -u origin claude/fix-balance-abc123

# 4. Create PR
curl -X POST https://pilot.grit.bot/api/repos/2/create_pr \
  -H "Content-Type: application/json" \
  -d '{"branch": "claude/fix-balance-abc123", "title": "Fix 500 error on /balance endpoint", "body": "Closes #XX"}'

# 5. Merge PR (user authorized deployment)
curl -X POST https://pilot.grit.bot/api/repos/2/merge_branch \
  -H "Content-Type: application/json" \
  -d '{"branch": "claude/fix-balance-abc123"}'

# 6. Pull to server
curl -X POST https://pilot.grit.bot/api/repos/2/pull

# 7. Find and restart service
curl "https://pilot.grit.bot/api/services?repo_id=2"  # Find service ID
curl -X POST https://pilot.grit.bot/api/services/1/restart

# 8. Report back to user
# "Fixed and deployed. CryptoBot service restarted. PR: https://github.com/..."
```

### Sync

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/repos/scan` | Sync issues from GitHub |
| POST | `/api/repos/sync_claude_md` | Sync CLAUDE.md to all repos |

**sync_claude_md parameters:**
```json
{
  "push": true,           // Push commits to remote (default: true)
  "repos": ["Repo1"],     // Sync to specific repos only (default: all)
  "dry_run": false        // Preview without making changes (default: false)
}
```

**Example: Sync CLAUDE.md to all repos**
```bash
curl -X POST https://pilot.grit.bot/api/repos/sync_claude_md \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Example: Preview sync (dry run)**
```bash
curl -X POST https://pilot.grit.bot/api/repos/sync_claude_md \
  -H "Content-Type: application/json" \
  -d '{"dry_run": true}'
```

**Example: Sync to specific repos only**
```bash
curl -X POST https://pilot.grit.bot/api/repos/sync_claude_md \
  -H "Content-Type: application/json" \
  -d '{"repos": ["CryptoBot", "GooseFlix"]}'
```

### System (GitPilot Self-Management)

GitPilot can monitor and update itself, pull updates, and restart automatically.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/system/status` | Get GitPilot version, uptime, update availability |
| POST | `/api/system/check-updates` | Check for updates on origin/main |
| POST | `/api/system/update` | Pull updates and restart GitPilot |
| POST | `/api/system/restart` | Restart GitPilot without pulling |
| GET | `/api/system/logs` | Get GitPilot's own logs |

**Check for updates:**
```bash
curl -X POST https://pilot.grit.bot/api/system/check-updates
```

**Response:**
```json
{
  "update_available": true,
  "current_commit": "abc1234",
  "remote_commit": "def5678",
  "commits_behind": 3
}
```

**Trigger update and restart:**
```bash
curl -X POST https://pilot.grit.bot/api/system/update
```

**Get system logs:**
```bash
curl "https://pilot.grit.bot/api/system/logs?lines=100"
```

---

## URLs

- **Dashboard**: https://pilot.grit.bot
- **API Base**: https://pilot.grit.bot/api

GitPilot syncs with GitHub every 60 seconds automatically.
