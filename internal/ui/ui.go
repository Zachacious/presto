package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Zachacious/presto/pkg/types"
	"github.com/briandowns/spinner"
)

// Colors for terminal output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[37m"
)

// UI handles user interface elements
type UI struct {
	spinner  *spinner.Spinner
	verbose  bool
	useColor bool
}

// New creates a new UI instance
func New(verbose bool) *UI {
	return &UI{
		verbose:  verbose,
		useColor: supportsColor(),
	}
}

// supportsColor checks if terminal supports color
func supportsColor() bool {
	term := os.Getenv("TERM")
	return term != "" && term != "dumb"
}

// colorize adds color to text if supported
func (ui *UI) colorize(color, text string) string {
	if !ui.useColor {
		return text
	}
	return color + text + ColorReset
}

// StartSpinner starts a spinner with message
func (ui *UI) StartSpinner(message string) {
	if ui.spinner != nil {
		ui.spinner.Stop()
	}

	ui.spinner = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	ui.spinner.Prefix = "🔄 "
	ui.spinner.Suffix = " " + message
	ui.spinner.Start()
}

// UpdateSpinner updates the spinner message
func (ui *UI) UpdateSpinner(message string) {
	if ui.spinner != nil {
		ui.spinner.Suffix = " " + message
	}
}

// StopSpinner stops the current spinner
func (ui *UI) StopSpinner() {
	if ui.spinner != nil {
		ui.spinner.Stop()
		ui.spinner = nil
	}
}

// Success prints a success message
func (ui *UI) Success(message string) {
	ui.StopSpinner()
	fmt.Printf("✅ %s\n", ui.colorize(ColorGreen, message))
}

// Error prints an error message
func (ui *UI) Error(message string) {
	ui.StopSpinner()
	fmt.Printf("❌ %s\n", ui.colorize(ColorRed, message))
}

// Warning prints a warning message
func (ui *UI) Warning(message string) {
	fmt.Printf("⚠️  %s\n", ui.colorize(ColorYellow, message))
}

// Info prints an info message
func (ui *UI) Info(message string) {
	fmt.Printf("ℹ️  %s\n", ui.colorize(ColorBlue, message))
}

// Progress prints a progress message (only if verbose)
func (ui *UI) Progress(message string) {
	if ui.verbose {
		fmt.Printf("🔄 %s\n", ui.colorize(ColorCyan, message))
	}
}

// ProcessingStart shows processing start info
func (ui *UI) ProcessingStart(fileCount int, mode types.ProcessingMode, model string) {
	ui.StopSpinner()

	modeText := string(mode)
	if mode == types.ModeTransform {
		modeText = "transform"
	} else if mode == types.ModeGenerate {
		modeText = "generate"
	}

	fmt.Printf("📁 Found %s to process\n",
		ui.colorize(ColorBlue, fmt.Sprintf("%d files", fileCount)))
	fmt.Printf("🧠 Using %s\n",
		ui.colorize(ColorPurple, model))
	fmt.Printf("🎯 Mode: %s\n",
		ui.colorize(ColorCyan, modeText))
	fmt.Println()
}

// FileProcessing shows file processing status
func (ui *UI) FileProcessing(filename string) {
	ui.StartSpinner(fmt.Sprintf("Processing %s...", filename))
}

// FileSuccess shows successful file processing
func (ui *UI) FileSuccess(inputFile, outputFile string, duration time.Duration, tokens int) {
	ui.StopSpinner()

	durationStr := fmt.Sprintf("%.1fs", duration.Seconds())
	tokensStr := ""
	if tokens > 0 {
		tokensStr = fmt.Sprintf(", %d tokens", tokens)
	}

	fmt.Printf("✅ %s → %s %s\n",
		ui.colorize(ColorGreen, shortenPath(inputFile)),
		ui.colorize(ColorBlue, shortenPath(outputFile)),
		ui.colorize(ColorGray, fmt.Sprintf("(%s%s)", durationStr, tokensStr)),
	)
}

// FileError shows failed file processing
func (ui *UI) FileError(inputFile string, err error) {
	ui.StopSpinner()
	fmt.Printf("❌ %s: %s\n",
		ui.colorize(ColorRed, shortenPath(inputFile)),
		ui.colorize(ColorRed, err.Error()),
	)
}

// FileSkipped shows skipped file
func (ui *UI) FileSkipped(inputFile string, reason string) {
	ui.StopSpinner()
	fmt.Printf("⏭️  %s %s\n",
		ui.colorize(ColorYellow, shortenPath(inputFile)),
		ui.colorize(ColorGray, fmt.Sprintf("(%s)", reason)),
	)
}

// Summary shows final processing summary
func (ui *UI) Summary(results []*types.ProcessingResult) {
	ui.StopSpinner()
	fmt.Println()

	// Calculate stats
	stats := ui.calculateStats(results)

	// Header
	fmt.Printf("🎉 %s\n", ui.colorize(ColorGreen, "Processing Complete!"))
	fmt.Println(strings.Repeat("=", 50))

	// Results
	if stats.Successful > 0 {
		successText := fmt.Sprintf("✅ %d files processed successfully", stats.Successful)
		if stats.Generated > 0 && stats.Transformed > 0 {
			successText += fmt.Sprintf(" (%d generated, %d transformed)", stats.Generated, stats.Transformed)
		} else if stats.Generated > 0 {
			successText += " (generated)"
		} else if stats.Transformed > 0 {
			successText += " (transformed)"
		}
		fmt.Printf("   %s\n", ui.colorize(ColorGreen, successText))
	}

	if stats.Skipped > 0 {
		fmt.Printf("   %s\n", ui.colorize(ColorYellow, fmt.Sprintf("⏭️  %d files skipped", stats.Skipped)))
	}

	if stats.Failed > 0 {
		fmt.Printf("   %s\n", ui.colorize(ColorRed, fmt.Sprintf("❌ %d files failed", stats.Failed)))
	}

	// Performance stats
	if stats.TotalTokens > 0 {
		fmt.Printf("   %s\n", ui.colorize(ColorPurple, fmt.Sprintf("🤖 %d AI tokens used", stats.TotalTokens)))
	}

	if stats.TotalDuration > 0 {
		fmt.Printf("   %s\n", ui.colorize(ColorBlue, fmt.Sprintf("⏱️  Total time: %s", formatDuration(stats.TotalDuration))))
	}

	if stats.EstimatedCost > 0 {
		fmt.Printf("   %s\n", ui.colorize(ColorCyan, fmt.Sprintf("💰 Estimated cost: $%.3f", stats.EstimatedCost)))
	}

	// Show failed files if any
	if stats.Failed > 0 && ui.verbose {
		fmt.Println()
		fmt.Printf("%s\n", ui.colorize(ColorRed, "❌ Failed files:"))
		for _, result := range results {
			if !result.Success && result.Error != nil {
				fmt.Printf("   • %s: %s\n",
					shortenPath(result.InputFile),
					ui.colorize(ColorRed, result.Error.Error()))
			}
		}
	}

	fmt.Println()
}

// ProcessingStats holds summary statistics
type ProcessingStats struct {
	Successful    int
	Failed        int
	Skipped       int
	Generated     int
	Transformed   int
	TotalTokens   int
	TotalDuration time.Duration
	EstimatedCost float64
}

// calculateStats computes processing statistics
func (ui *UI) calculateStats(results []*types.ProcessingResult) ProcessingStats {
	stats := ProcessingStats{}

	for _, result := range results {
		if result.Skipped {
			stats.Skipped++
		} else if result.Success {
			stats.Successful++
			stats.TotalTokens += result.AITokensUsed
			stats.TotalDuration += result.Duration

			if result.Mode == types.ModeGenerate {
				stats.Generated++
			} else {
				stats.Transformed++
			}
		} else {
			stats.Failed++
		}
	}

	// Rough cost estimation (adjust based on your model pricing)
	// GPT-4: ~$0.03/1K tokens, GPT-3.5: ~$0.002/1K tokens
	stats.EstimatedCost = float64(stats.TotalTokens) * 0.00003 // Rough GPT-4 estimate

	return stats
}

// shortenPath shortens file paths for display
func shortenPath(path string) string {
	if len(path) <= 50 {
		return path
	}

	// Show just the filename if path is too long
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		return "..." + parts[len(parts)-1]
	}

	return path[:47] + "..."
}

// formatDuration formats duration for display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// FileContinuation shows continuation attempt
func (ui *UI) FileContinuation(filename string, attempt, maxAttempts int) {
	ui.UpdateSpinner(fmt.Sprintf("Processing %s... (continuation %d/%d)",
		filename, attempt, maxAttempts))
}

// Warning about potential incompleteness
func (ui *UI) FileIncompleteWarning(filename string, attempts int) {
	ui.Warning(fmt.Sprintf("File %s may be incomplete after %d continuation attempts",
		filename, attempts))
}
