# Story 5.1: Add Link Checker to CI

Status: done

<!-- ═══════════════════════════════════════════════════════════════════════════ -->
<!-- BEADS TRACKING -->
<!-- ═══════════════════════════════════════════════════════════════════════════ -->

**Beads IDs:**

- Epic: `bd-yip`
- Story: `bd-yip.1`

**Quick Commands:**

- View tasks: `bd list --parent bd-yip.1`
- Find ready work: `bd ready --parent bd-yip.1`
- Mark task done: `bd close <task_id>`

## Story

As a **maintainer reviewing PRs**,
I want **automated link checking in CI**,
So that **broken links are caught before merge**.

## Acceptance Criteria

1. **AC1: Link checker integration**
   - **Given** deploy-docs.yml workflow exists with build steps
   - **When** lychee link checker step is added after npm build
   - **Then** link validation runs on push to docs/docusaurus-site branch

2. **AC2: Internal link failures block deployment**
   - **Given** a documentation file contains a broken internal link
   - **When** CI workflow runs
   - **Then** workflow fails with clear error message showing broken link

3. **AC3: External link warnings (non-blocking)**
   - **Given** a documentation file contains a broken external link
   - **When** CI workflow runs
   - **Then** workflow shows warning but continues (external sites may be temporarily down)

4. **AC4: Results visible in GitHub Actions**
   - **Given** a push triggers the workflow
   - **When** link checker runs
   - **Then** results are visible in GitHub Actions summary with links count

5. **AC5: Minimal configuration (no external config file)**
   - **Given** lychee runs in CI
   - **When** configured via inline options
   - **Then** no separate .lychee.toml or .lycheeignore file is required

## Tasks / Subtasks

<!-- ═══════════════════════════════════════════════════════════════════════════ -->
<!-- BEADS IS AUTHORITATIVE: Task status is tracked in Beads, not checkboxes.   -->
<!-- ═══════════════════════════════════════════════════════════════════════════ -->

- [x] **Task 1: Research lychee configuration** (AC: 5) `bd-yip.1.1`
  - [x] 1.1: Verify latest lychee-action version (check GitHub releases)
  - [x] 1.2: Confirm `--cache` option availability for CI performance
  - [x] 1.3: Identify required exclude patterns for Docusaurus sites

- [x] **Task 2: Add lychee step to deploy-docs.yml** (AC: 1, 2, 3) `bd-yip.1.2`
  - [x] 2.1: Add lychee-action step after "Build website" step
  - [x] 2.2: Configure two-step approach: internal (fail) + external (warn)
  - [x] 2.3: Add cache configuration for performance
  - [x] 2.4: Set timeout (30s) and excludes (localhost, edit links, mailto)

- [x] **Task 3: Validate link checker** (AC: 4) `bd-yip.1.3`
  - [x] 3.1: Push change and verify workflow runs
  - [x] 3.2: Verify GitHub Actions summary shows link count
  - [x] 3.3: Optionally test with intentional broken link

- [x] **Task 4: Update CONTRIBUTING.md** (AC: 5) `bd-yip.1.4`
  - [x] 4.1: Add "CI Quality Gates" section describing link checker
  - [x] 4.2: Document how to run lychee locally for pre-push validation

---

## CRITICAL: Implementation Requirements

### File to Modify

`.github/workflows/deploy-docs.yml` — add after "Build website" step (line 46)

### Recommended Implementation (Two-Step Approach)

```yaml
      - name: Check internal links
        uses: lycheeverse/lychee-action@v2
        with:
          args: |
            --verbose
            --no-progress
            --offline
            --include-fragments
            --timeout 30
            --exclude-path 'website/build/search/**'
            website/build
          fail: true

      - name: Check external links (non-blocking)
        uses: lycheeverse/lychee-action@v2
        continue-on-error: true
        with:
          args: |
            --verbose
            --no-progress
            --timeout 30
            --scheme https
            --scheme http
            --exclude 'localhost'
            --exclude '127.0.0.1'
            --exclude 'tree/docs/docusaurus-site'
            --exclude 'example.com'
            --exclude 'mailto:'
            website/build
```

### Expected Output

**Success:**
```
✅ 0 errors in 234 links
```

**Failure (internal broken link):**
```
❌ Error: website/build/docs/missing-page/index.html
   Broken link: /docs/nonexistent
   Status: 404 Not Found
```

### Required Exclude Patterns

| Pattern | Reason |
|---------|--------|
| `localhost`, `127.0.0.1` | Local dev URLs |
| `edit/docs/docusaurus-site` | GitHub edit links (dynamic) |
| `example.com` | Placeholder URLs in docs |
| `mailto:` | Email links don't resolve |
| `website/build/search/**` | Search index JSON files |

---

## Architecture Compliance

**PRD:** FR17 (zero broken links), NFR10 (100% link validity)
**Architecture:** CI/CD Quality Gates section

---

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Completion Notes List

- Implemented two-step link checking approach: internal links (blocking) + external links (non-blocking)
- Used lychee-action v2 for latest features and stability
- Internal link check uses `--offline` flag for fast, accurate local file validation
- External link check excludes common false positives (localhost, edit links, example.com, mailto:)
- Added "CI Quality Gates" section to CONTRIBUTING.md with local lychee usage instructions
- Website build validated locally - passes without errors

### Code Review Fixes (2025-12-30)

- **H1 Fixed**: Added `cache: true` to external links step for CI performance
- **H2 Fixed**: Updated CONTRIBUTING.md local commands to match CI (added --timeout 30, --exclude-path, all excludes)
- **M2 Fixed**: Corrected story template exclude pattern from `edit/` to `tree/` (GitHub file browser URLs)

### File List

- `.github/workflows/deploy-docs.yml` - Added lychee link checker steps (internal + external)
- `CONTRIBUTING.md` - Added "CI Quality Gates (Documentation)" section
