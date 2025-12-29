package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/types"
)

var resolveConflictsCmd = &cobra.Command{
	Use:   "resolve-conflicts",
	Short: "Resolve git merge conflicts in JSONL files",
	Long: `Detect and resolve git merge conflicts in .beads/issues.jsonl.

Modes:
  - Detection only (default): Show conflicts without modifying files
  - Auto-resolve: Mechanically resolve by remapping conflicting IDs
  - Interactive: Review each conflict (future)

The mechanical resolution strategy:
  1. Keep all HEAD issues unchanged
  2. Remap BASE issues with conflicting IDs to new IDs
  3. Update all text references and dependencies

Example:
  bd resolve-conflicts              # Show conflicts
  bd resolve-conflicts --auto       # Auto-resolve conflicts
  bd resolve-conflicts --dry-run    # Preview resolution`,
	Run: func(cmd *cobra.Command, _ []string) {
		auto, _ := cmd.Flags().GetBool("auto")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		ctx := context.Background()

		// Find JSONL file path
		jsonlPath := findJSONLPath()

		// Detect conflicts
		conflicts, err := detectConflicts(jsonlPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting conflicts: %v\n", err)
			os.Exit(1)
		}

		if len(conflicts) == 0 {
			if !jsonOutput {
				fmt.Println("✓ No conflicts found in", jsonlPath)
			} else {
				outputJSON(map[string]interface{}{
					"conflicts": 0,
					"file":      jsonlPath,
				})
			}
			return
		}

		// Show conflicts
		if !jsonOutput {
			fmt.Printf("Found %d conflict(s) in %s:\n\n", len(conflicts), jsonlPath)
			for i, conflict := range conflicts {
				fmt.Printf("Conflict %d (lines %d-%d):\n", i+1, conflict.LineStart, conflict.LineEnd)
				fmt.Printf("  HEAD issues: %d\n", len(conflict.HeadIssues))
				for _, issue := range conflict.HeadIssues {
					fmt.Printf("    - %s: %s\n", color.CyanString(issue.ID), issue.Title)
				}
				fmt.Printf("  BASE issues: %d\n", len(conflict.BaseIssues))
				for _, issue := range conflict.BaseIssues {
					fmt.Printf("    - %s: %s\n", color.YellowString(issue.ID), issue.Title)
				}
				fmt.Println()
			}
		}

		if !auto && !dryRun {
			if !jsonOutput {
				fmt.Println("Run 'bd resolve-conflicts --auto' to apply automatic resolution.")
			} else {
				outputJSON(map[string]interface{}{
					"conflicts": len(conflicts),
					"file":      jsonlPath,
					"details":   formatConflictsJSON(conflicts),
				})
			}
			return
		}

		// Resolve conflicts
		resolutions, err := resolveConflictsMechanical(ctx, conflicts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving conflicts: %v\n", err)
			os.Exit(1)
		}

		if !jsonOutput {
			fmt.Println("Proposed resolutions:")
			for _, res := range resolutions {
				switch res.Action {
				case "keep":
					fmt.Printf("  ✓ Keep %s unchanged\n", color.CyanString(res.IssueID))
				case "remap":
					fmt.Printf("  ↻ Remap %s → %s (%s)\n",
						color.YellowString(res.OldID),
						color.GreenString(res.NewID),
						res.Reason)
				}
			}
			fmt.Println()
		}

		if dryRun {
			if !jsonOutput {
				fmt.Println("Dry-run mode: no changes made")
			} else {
				outputJSON(map[string]interface{}{
					"dry_run":     true,
					"resolutions": formatResolutionsJSON(resolutions),
				})
			}
			return
		}

		// Apply resolutions
		if err := applyResolutions(ctx, jsonlPath, conflicts, resolutions); err != nil {
			fmt.Fprintf(os.Stderr, "Error applying resolutions: %v\n", err)
			os.Exit(1)
		}

		if !jsonOutput {
			fmt.Printf("✓ Resolved %d conflict(s)\n", len(conflicts))
			fmt.Println("\nNext steps:")
			fmt.Println("  1. Review changes: git diff", filepath.Base(jsonlPath))
			fmt.Println("  2. Import to database: bd import")
			fmt.Println("  3. Commit resolution: git add", filepath.Base(jsonlPath), "&& git commit")
		} else {
			outputJSON(map[string]interface{}{
				"success":     true,
				"conflicts":   len(conflicts),
				"resolutions": len(resolutions),
				"file":        jsonlPath,
			})
		}
	},
}

func init() {
	resolveConflictsCmd.Flags().Bool("auto", false, "Automatically resolve conflicts")
	resolveConflictsCmd.Flags().Bool("dry-run", false, "Show what would be resolved without making changes")
	rootCmd.AddCommand(resolveConflictsCmd)
}

// ConflictBlock represents a git merge conflict section
type ConflictBlock struct {
	HeadIssues []types.Issue
	BaseIssues []types.Issue
	LineStart  int
	LineEnd    int
}

// Resolution represents how to resolve a conflict
type Resolution struct {
	Action   string // "keep", "remap"
	IssueID  string // For "keep" action
	OldID    string // For "remap" action
	NewID    string // For "remap" action
	Reason   string
}

// detectConflicts parses a JSONL file and finds git conflict markers
func detectConflicts(jsonlPath string) ([]ConflictBlock, error) {
	file, err := os.Open(jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", jsonlPath, err)
	}
	defer file.Close()

	var conflicts []ConflictBlock
	var current *ConflictBlock
	inConflict := false
	inHead := false
	lineNum := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNum++

		// Skip empty lines
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "<<<<<<<"):
			// Start of conflict
			inConflict = true
			inHead = true
			current = &ConflictBlock{
				LineStart: lineNum,
			}

		case strings.HasPrefix(line, "======="):
			// Switch from HEAD to base
			if inConflict {
				inHead = false
			}

		case strings.HasPrefix(line, ">>>>>>>"):
			// End of conflict
			if inConflict {
				inConflict = false
				current.LineEnd = lineNum
				conflicts = append(conflicts, *current)
				current = nil
			}

		case inConflict && !strings.HasPrefix(line, "<") && !strings.HasPrefix(line, "=") && !strings.HasPrefix(line, ">"):
			// Parse issue line (must be valid JSON)
			var issue types.Issue
			if err := json.Unmarshal([]byte(line), &issue); err != nil {
				// Skip non-JSON lines in conflict
				continue
			}

			if inHead {
				current.HeadIssues = append(current.HeadIssues, issue)
			} else {
				current.BaseIssues = append(current.BaseIssues, issue)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return conflicts, nil
}

// resolveConflictsMechanical generates mechanical resolutions (no AI)
func resolveConflictsMechanical(ctx context.Context, conflicts []ConflictBlock) ([]Resolution, error) {
	var resolutions []Resolution

	// Collect all HEAD issue IDs (these are kept)
	headIDs := make(map[string]bool)
	for _, conflict := range conflicts {
		for _, issue := range conflict.HeadIssues {
			headIDs[issue.ID] = true
			resolutions = append(resolutions, Resolution{
				Action:  "keep",
				IssueID: issue.ID,
				Reason:  "HEAD version",
			})
		}
	}

	// For BASE issues, remap if ID collides with HEAD
	for _, conflict := range conflicts {
		for _, issue := range conflict.BaseIssues {
			if headIDs[issue.ID] {
				// ID collision: remap to new ID
				newID, err := getNextAvailableID(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to get next ID: %w", err)
				}
				resolutions = append(resolutions, Resolution{
					Action: "remap",
					OldID:  issue.ID,
					NewID:  newID,
					Reason: fmt.Sprintf("ID %s exists in both HEAD and BASE", issue.ID),
				})
				headIDs[newID] = true // Reserve the new ID
			} else {
				// No collision: keep as-is
				resolutions = append(resolutions, Resolution{
					Action:  "keep",
					IssueID: issue.ID,
					Reason:  "No collision",
				})
				headIDs[issue.ID] = true
			}
		}
	}

	return resolutions, nil
}

// applyResolutions writes the resolved JSONL file
func applyResolutions(ctx context.Context, jsonlPath string, conflicts []ConflictBlock, resolutions []Resolution) error {
	// Read entire file
	content, err := os.ReadFile(jsonlPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Build remap table
	remapTable := make(map[string]string)
	for _, res := range resolutions {
		if res.Action == "remap" {
			remapTable[res.OldID] = res.NewID
		}
	}

	// Process line by line
	var resolved []string
	inConflict := false
	var conflictLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "<<<<<<<") {
			inConflict = true
			conflictLines = []string{}
			continue
		}

		if strings.HasPrefix(trimmed, ">>>>>>>") {
			inConflict = false
			// Append resolved conflict lines
			resolved = append(resolved, conflictLines...)
			conflictLines = nil
			continue
		}

		if strings.HasPrefix(trimmed, "=======") {
			continue
		}

		if inConflict {
			// Process issue line
			if trimmed != "" {
				var issue types.Issue
				if err := json.Unmarshal([]byte(trimmed), &issue); err == nil {
					// Remap ID if needed
					if newID, ok := remapTable[issue.ID]; ok {
						issue.ID = newID
					}

					// Remap dependencies
					for i, dep := range issue.Dependencies {
						if newID, ok := remapTable[dep.DependsOnID]; ok {
							issue.Dependencies[i].DependsOnID = newID
						}
					}

					// Remap text references in description, design, acceptance, notes
					issue.Description = remapTextReferences(issue.Description, remapTable)
					issue.Design = remapTextReferences(issue.Design, remapTable)
					issue.AcceptanceCriteria = remapTextReferences(issue.AcceptanceCriteria, remapTable)
					issue.Notes = remapTextReferences(issue.Notes, remapTable)

					// Re-serialize
					jsonBytes, _ := json.Marshal(issue)
					conflictLines = append(conflictLines, string(jsonBytes))
				}
			}
		} else {
			// Outside conflict: check if we need to remap references
			if trimmed != "" && !strings.HasPrefix(trimmed, "<") && !strings.HasPrefix(trimmed, "=") && !strings.HasPrefix(trimmed, ">") {
				var issue types.Issue
				if err := json.Unmarshal([]byte(trimmed), &issue); err == nil {
					modified := false

					// Remap dependencies
					for i, dep := range issue.Dependencies {
						if newID, ok := remapTable[dep.DependsOnID]; ok {
							issue.Dependencies[i].DependsOnID = newID
							modified = true
						}
					}

					// Remap text references
					newDesc := remapTextReferences(issue.Description, remapTable)
					if newDesc != issue.Description {
						issue.Description = newDesc
						modified = true
					}

					newDesign := remapTextReferences(issue.Design, remapTable)
					if newDesign != issue.Design {
						issue.Design = newDesign
						modified = true
					}

					newAcceptance := remapTextReferences(issue.AcceptanceCriteria, remapTable)
					if newAcceptance != issue.AcceptanceCriteria {
						issue.AcceptanceCriteria = newAcceptance
						modified = true
					}

					newNotes := remapTextReferences(issue.Notes, remapTable)
					if newNotes != issue.Notes {
						issue.Notes = newNotes
						modified = true
					}

					if modified {
						jsonBytes, _ := json.Marshal(issue)
						resolved = append(resolved, string(jsonBytes))
					} else {
						resolved = append(resolved, line)
					}
				} else {
					resolved = append(resolved, line)
				}
			} else {
				resolved = append(resolved, line)
			}
		}
	}

	// Write back
	output := strings.Join(resolved, "\n")
	if err := os.WriteFile(jsonlPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Helper functions

func getNextAvailableID(ctx context.Context) (string, error) {
	if store == nil {
		return "", fmt.Errorf("store not initialized")
	}

	// Get current max ID
	allIssues, err := store.SearchIssues(ctx, "", types.IssueFilter{})
	if err != nil {
		return "", err
	}

	maxNum := 0
	prefix := ""

	for _, issue := range allIssues {
		// Parse ID format: prefix-number
		parts := strings.Split(issue.ID, "-")
		if len(parts) >= 2 {
			if prefix == "" {
				prefix = strings.Join(parts[:len(parts)-1], "-")
			}
			var num int
			fmt.Sscanf(parts[len(parts)-1], "%d", &num)
			if num > maxNum {
				maxNum = num
			}
		}
	}

	if prefix == "" {
		prefix = "bd"
	}

	return fmt.Sprintf("%s-%d", prefix, maxNum+1), nil
}

func remapTextReferences(text string, remapTable map[string]string) string {
	result := text
	for oldID, newID := range remapTable {
		result = strings.ReplaceAll(result, oldID, newID)
	}
	return result
}

func formatConflictsJSON(conflicts []ConflictBlock) []map[string]interface{} {
	var result []map[string]interface{}
	for _, conflict := range conflicts {
		headIDs := make([]string, len(conflict.HeadIssues))
		for i, issue := range conflict.HeadIssues {
			headIDs[i] = issue.ID
		}
		baseIDs := make([]string, len(conflict.BaseIssues))
		for i, issue := range conflict.BaseIssues {
			baseIDs[i] = issue.ID
		}
		result = append(result, map[string]interface{}{
			"line_start":  conflict.LineStart,
			"line_end":    conflict.LineEnd,
			"head_issues": headIDs,
			"base_issues": baseIDs,
		})
	}
	return result
}

func formatResolutionsJSON(resolutions []Resolution) []map[string]interface{} {
	var result []map[string]interface{}
	for _, res := range resolutions {
		r := map[string]interface{}{
			"action": res.Action,
			"reason": res.Reason,
		}
		if res.Action == "keep" {
			r["issue_id"] = res.IssueID
		} else if res.Action == "remap" {
			r["old_id"] = res.OldID
			r["new_id"] = res.NewID
		}
		result = append(result, r)
	}
	return result
}
