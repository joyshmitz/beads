# Story 5.2: Add Token Count Validation

Status: done

<!-- ═══════════════════════════════════════════════════════════════════════════ -->
<!-- BEADS TRACKING -->
<!-- ═══════════════════════════════════════════════════════════════════════════ -->

**Beads IDs:**

- Epic: `bd-yip`
- Story: `bd-yip.2`

**Quick Commands:**

- View tasks: `bd list --parent bd-yip.2`
- Find ready work: `bd ready --parent bd-yip.2`
- Mark task done: `bd close <task_id>`

## Story

As a **maintainer protecting token budget**,
I want **automated token count validation**,
So that **llms-full.txt stays under 50K tokens**.

## Acceptance Criteria

1. **AC1: Word count validation step**
   - **Given** deploy-docs.yml has no token validation
   - **When** I add word count check step
   - **Then** CI fails if llms-full.txt exceeds 37,500 words (~50K tokens)

2. **AC2: Current count reported**
   - **Given** word count validation runs
   - **When** CI completes
   - **Then** current word count is reported in CI output

3. **AC3: Check timing**
   - **Given** llms-full.txt is generated
   - **When** word count check runs
   - **Then** check runs after llms-full.txt generation step

## Tasks / Subtasks

<!-- ═══════════════════════════════════════════════════════════════════════════ -->
<!-- BEADS IS AUTHORITATIVE: Task status is tracked in Beads, not checkboxes.   -->
<!-- ═══════════════════════════════════════════════════════════════════════════ -->

- [x] **Task 1: Add word count validation step to deploy-docs.yml** (AC: 1, 2, 3)
  - [x] 1.1: Add bash step after "Generate llms-full.txt" step
  - [x] 1.2: Count words using `wc -w`
  - [x] 1.3: Compare against threshold (37500 words)
  - [x] 1.4: Output current count and fail if exceeded

- [x] **Task 2: Test validation locally**
  - [x] 2.1: Run word count on current llms-full.txt (15,885 words)
  - [x] 2.2: Verify threshold is appropriate (OK, well under 37,500)

---

## Dev Notes

### Architecture Compliance

**PRD Requirements:**
- **FR19:** CI validates token budget for llms-full.txt
- **NFR11:** Token budget: <50K tokens (~37,500 words)

### Technical Context

**Current deploy-docs.yml structure:**
```yaml
jobs:
  build:
    steps:
      - Checkout
      - Setup Node.js
      - Install dependencies
      - Generate llms-full.txt    # <-- Add validation AFTER this
      - Build website
      - Check internal links
      - Check external links
      - Upload artifact
```

### Implementation Approach

Simple bash step with word count:

```yaml
      - name: Validate token budget
        run: |
          WORD_COUNT=$(wc -w < website/static/llms-full.txt)
          echo "📊 llms-full.txt word count: $WORD_COUNT / 37500"
          if [ "$WORD_COUNT" -gt 37500 ]; then
            echo "❌ Token budget exceeded! Max: 37500 words (~50K tokens)"
            exit 1
          fi
          echo "✅ Token budget OK"
```

---

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Completion Notes List

- Added "Validate token budget" step to deploy-docs.yml after llms-full.txt generation
- Uses `wc -w` to count words (simple, portable)
- Threshold: 37,500 words (~50K tokens per NFR11)
- Current count: 15,885 words (42% of budget) - plenty of headroom
- Clear emoji output for CI logs

### File List

- `.github/workflows/deploy-docs.yml` - Added token budget validation step

---

## Senior Developer Review (AI)

**Reviewer:** Claude Opus 4.5 | **Date:** 2025-12-30

### Review Outcome: ✅ APPROVED (with minor notes)

### Findings

| # | Severity | Issue | Resolution |
|---|----------|-------|------------|
| 1 | LOW | Token-to-word ratio (1.33 tokens/word) is approximate | Acceptable - current count (15,885) well under threshold |
| 2 | MEDIUM | Missing PR trigger for validation | **FIXED** - Added `pull_request` trigger to workflow |

### Code Quality
- ✅ Implementation follows simple, portable bash approach
- ✅ Clear emoji-based output for CI logs
- ✅ Threshold correctly set to 37,500 words per NFR11

### AC Verification
- ✅ **AC1:** Word count validation step added
- ✅ **AC2:** Current count reported in CI output
- ✅ **AC3:** Check runs after llms-full.txt generation (and now on PR too)

### Changes Made During Review
1. Added `pull_request` trigger to workflow (AC3 compliance)
2. Added `if: github.event_name == 'push'` condition to deploy job

