package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/spf13/cobra"
)

var (
	doneConfidence int
	donePurpose    string
	doneFiles      string
	doneTests      string
	doneReproduce  []string
	doneProof      string
	doneReasoning  string
	doneCtxStdin   bool
)

var doneCmd = &cobra.Command{
	Use:   "done <ref>",
	Short: "Complete an item with review context",
	Long: `Complete an item and move it to done with structured review context.

This is the agent-facing way to mark work complete. It wraps
CompleteItemWithContext and sets up the item for human review.

Review context can be provided via flags or as JSON on stdin (--context-stdin).

Examples:
  ob done 34 --confidence 45 --purpose "Replace cookie sessions with JWT"
  echo '{"purpose":"..."}' | ob done 34 --confidence 45 --context-stdin`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref := args[0]

		if doneConfidence < 0 || doneConfidence > 100 {
			return fmt.Errorf("--confidence must be between 0 and 100, got %d", doneConfidence)
		}

		ctx, err := buildReviewContext(cmd)
		if err != nil {
			return err
		}

		eng, err := getEngine()
		if err != nil {
			return err
		}

		if err := eng.CompleteItemWithContext(ref, ctx, doneConfidence, getUserID(), getSessionID()); err != nil {
			return err
		}

		fmt.Printf("Completed #%s with confidence %d%% — queued for human review\n", ref, doneConfidence)
		return nil
	},
}

func buildReviewContext(cmd *cobra.Command) (domain.ReviewContext, error) {
	if doneCtxStdin {
		return readContextFromStdin()
	}
	return buildContextFromFlags()
}

func readContextFromStdin() (domain.ReviewContext, error) {
	var ctx domain.ReviewContext
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return ctx, fmt.Errorf("failed to read context from stdin: %w", err)
	}
	if len(data) == 0 {
		return ctx, fmt.Errorf("--context-stdin specified but stdin is empty")
	}
	if err := json.Unmarshal(data, &ctx); err != nil {
		return ctx, fmt.Errorf("invalid JSON on stdin: %w", err)
	}
	return ctx, nil
}

func buildContextFromFlags() (domain.ReviewContext, error) {
	ctx := domain.ReviewContext{
		Purpose:   donePurpose,
		Reproduce: doneReproduce,
		Reasoning: doneReasoning,
	}

	if doneFiles != "" {
		files, err := parseFileChanges(doneFiles)
		if err != nil {
			return ctx, err
		}
		ctx.FilesChanged = files
	}

	if doneTests != "" {
		ctx.TestsWritten = parseTestResults(doneTests)
	}

	if doneProof != "" {
		ctx.Proof = parseProofItems(doneProof)
	}

	return ctx, nil
}

// parseFileChanges parses "path:+added-removed,path2:+added-removed"
func parseFileChanges(raw string) ([]domain.FileChange, error) {
	var files []domain.FileChange
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 2)
		fc := domain.FileChange{Path: parts[0]}
		if len(parts) == 2 {
			counts := parts[1]
			added, removed, err := parseAddedRemoved(counts)
			if err != nil {
				return nil, fmt.Errorf("invalid file change %q: %w", entry, err)
			}
			fc.Added = added
			fc.Removed = removed
		}
		files = append(files, fc)
	}
	return files, nil
}

// parseAddedRemoved parses "+82-41" into (82, 41)
func parseAddedRemoved(s string) (int, int, error) {
	s = strings.TrimPrefix(s, "+")
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected +N-N format, got %q", s)
	}
	added, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid added count %q: %w", parts[0], err)
	}
	removed, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid removed count %q: %w", parts[1], err)
	}
	return added, removed, nil
}

// parseTestResults parses "TestName:pass,TestName2:fail"
func parseTestResults(raw string) []domain.TestResult {
	var results []domain.TestResult
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 2)
		tr := domain.TestResult{Name: parts[0]}
		if len(parts) == 2 {
			tr.Passed = strings.EqualFold(parts[1], "pass")
		}
		results = append(results, tr)
	}
	return results
}

// parseProofItems parses "check:status,check:status:detail"
func parseProofItems(raw string) []domain.ProofItem {
	var items []domain.ProofItem
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 3)
		pi := domain.ProofItem{Check: parts[0]}
		if len(parts) >= 2 {
			pi.Status = parts[1]
		}
		if len(parts) == 3 {
			pi.Detail = parts[2]
		}
		items = append(items, pi)
	}
	return items
}

func init() {
	doneCmd.Flags().IntVar(&doneConfidence, "confidence", 0, "confidence level 0-100 (required)")
	doneCmd.Flags().StringVar(&donePurpose, "purpose", "", "purpose of the change")
	doneCmd.Flags().StringVar(&doneFiles, "files", "", "files changed (path:+added-removed,...)")
	doneCmd.Flags().StringVar(&doneTests, "tests", "", "test results (name:pass|fail,...)")
	doneCmd.Flags().StringArrayVar(&doneReproduce, "reproduce", nil, "reproduction commands (repeatable)")
	doneCmd.Flags().StringVar(&doneProof, "proof", "", "proof items (check:status[:detail],...)")
	doneCmd.Flags().StringVar(&doneReasoning, "reasoning", "", "agent decision rationale")
	doneCmd.Flags().BoolVar(&doneCtxStdin, "context-stdin", false, "read ReviewContext JSON from stdin")
	if err := doneCmd.MarkFlagRequired("confidence"); err != nil {
		panic(fmt.Sprintf("failed to mark confidence flag required: %v", err))
	}
	rootCmd.AddCommand(doneCmd)
}
