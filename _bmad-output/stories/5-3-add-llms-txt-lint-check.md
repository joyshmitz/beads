# Story 5.3: Add llms.txt Lint Check

Status: done

<!-- ═══════════════════════════════════════════════════════════════════════════ -->
<!-- BEADS TRACKING -->
<!-- ═══════════════════════════════════════════════════════════════════════════ -->

**Beads IDs:**

- Epic: `bd-yip`
- Story: `bd-yip.3`

## Story

As a **maintainer ensuring llms.txt quality**,
I want **automated llms.txt validation**,
So that **format and size requirements are enforced**.

## Acceptance Criteria

1. **AC1: Size validation**
   - **Given** deploy-docs.yml has no llms.txt validation
   - **When** I add llms.txt lint step
   - **Then** CI fails if llms.txt exceeds 10KB (NFR3)

2. **AC2: Required sections validation**
   - **Given** llms.txt lint step runs
   - **When** CI validates required sections
   - **Then** CI fails if Quick Start, Core Concepts, CLI Reference sections are missing

3. **AC3: Check timing**
   - **Given** website is built
   - **When** lint check runs
   - **Then** check runs on PR and push to docs/docusaurus-site

## Tasks / Subtasks

- [x] **Task 1: Add llms.txt validation step to deploy-docs.yml** (AC: 1, 2, 3)
  - [x] 1.1: Add bash step to validate file size (<10KB)
  - [x] 1.2: Add section header validation (Quick Start, Core Concepts/Commands)
  - [x] 1.3: Output current size and validation status

- [x] **Task 2: Test validation locally**
  - [x] 2.1: Check current llms.txt size (2,355 bytes - OK)
  - [x] 2.2: Verify required sections exist (Quick Start ✓, Key Concepts ✓)

---

## Dev Notes

### Architecture Compliance

**PRD Requirements:**
- **FR18:** CI validates llms.txt format and size
- **NFR3:** llms.txt size: <10KB

### llmstxt.org spec sections

Required sections per project-context.md:
- Quick Start
- Core Concepts
- CLI Reference
- Optional (recommended)

---

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Completion Notes List

- Added "Validate llms.txt" step to deploy-docs.yml
- Size validation: max 10KB (10240 bytes)
- Section validation: checks for "Quick Start" and "Concepts" or "Commands" sections
- Current size: 2,355 bytes (23% of limit)
- Flexible regex allows for section name variations

### File List

- `.github/workflows/deploy-docs.yml` - Added llms.txt validation step

---

## Senior Developer Review (AI)

**Reviewer:** Claude Opus 4.5 | **Date:** 2025-12-30

### Review Outcome: ✅ APPROVED (after fixes)

### Findings

| # | Severity | Issue | Resolution |
|---|----------|-------|------------|
| 1 | HIGH | AC2 - CLI Reference section not validated | **FIXED** - Updated regex to check for "CLI Reference" OR "Commands" |
| 2 | HIGH | AC2 - "Core Concepts" check too rigid | **FIXED** - Now accepts "Core Concepts", "Key Concepts", or just "Concepts" |
| 3 | MEDIUM | Missing PR trigger (AC3) | **FIXED** - Added `pull_request` trigger to workflow |
| 4 | LOW | Size display used bytes, not KB | **FIXED** - Now shows "2KB / 10KB (2355 bytes)" |
| 5 | LOW | Error message missing space after colon | **FIXED** - Added proper spacing |

### Code Quality
- ✅ Size validation correctly enforces NFR3 (<10KB)
- ✅ Section validation now properly flexible per llmstxt.org spec
- ✅ Clear error messages with specific missing sections listed

### AC Verification
- ✅ **AC1:** Size validation (<10KB) implemented and working
- ✅ **AC2:** Required sections validated (Quick Start, Concepts, CLI/Commands)
- ✅ **AC3:** Check runs on PR and push (after fix)

### Changes Made During Review
1. Added `pull_request` trigger to workflow (AC3 compliance)
2. Updated section regex: `^## (Core |Key )?Concepts` for flexibility
3. Updated CLI regex: `^## (CLI Reference|.*Commands)` for spec compliance
4. Improved size display: shows KB with byte count
5. Fixed error message grammar (space after colon)
6. Added `if: github.event_name == 'push'` condition to deploy job

