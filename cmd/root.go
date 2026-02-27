package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/dominionthedev/logfmt/internal/formatter"
)

var opts formatter.Options

var rootCmd = &cobra.Command{
	Use:   `logfmt`,
	Short: `🪵 Pipe logs and get formatted, colorized, filterable output`,
	Long:  `logfmt reads log lines from stdin and renders them in a clean, colorized terminal format. Supports JSON and logfmt-style logs. Falls back gracefully for plain text.`,
	Example: `  tail -f app.log | logfmt
  cat app.log | logfmt --level warn
  kubectl logs my-pod | logfmt --filter "user_id=42"
  cat app.log | logfmt --time-only
  tail -f app.log | logfmt --level error --filter "database"`,

	RunE: func(cmd *cobra.Command, args []string) error {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			// Nothing piped — print help
			return cmd.Help()
		}

		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB lines

		for scanner.Scan() {
			line := scanner.Text()
			out := formatter.FormatLine(line, opts)
			if out != "" {
				fmt.Println(out)
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "logfmt: read error: %v\n", err)
			return err
		}

		return nil
	},
}

func Execute(version string) {
	// Set the version for the root command
	rootCmd.Version = version
	// fang.Execute now expects a context and returns an error
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}


func init() {
	rootCmd.Flags().StringVarP(&opts.Filter, "filter", "f", "", "Only show lines containing this string (case-insensitive)")
	rootCmd.Flags().StringVarP(&opts.LevelMin, "level", "l", "", "Minimum log level to show (debug|info|warn|error|fatal)")
	rootCmd.Flags().BoolVar(&opts.NoColor, "no-color", false, "Disable color output")
	rootCmd.Flags().BoolVarP(&opts.TimeOnly, "time-only", "t", false, "Show only time, level and message — hide all KV pairs")
}
