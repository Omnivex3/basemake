package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/DynamicKarabo/basemake/internal/display"
	"github.com/DynamicKarabo/basemake/internal/profile"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage and view query execution profiles",
	Long: `Browse the execution history of your queries.
The profile engine automatically records timings and execution plans for queries you run.

  basemake profile list
  basemake profile view <hash>
  basemake profile clear
  basemake profile clear <hash>`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiled queries",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := profile.ProfileDir()
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintln(os.Stdout, "No profiled queries yet.")
				return nil
			}
			return err
		}

		type profileSummary struct {
			Hash          string
			NormalizedSQL string
			RunCount      int
			AvgDuration   int64
			LastRun       string
			LastRunTime   int64
			PlanHash      string
		}

		var summaries []profileSummary

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}

			hash := strings.TrimSuffix(entry.Name(), ".json")
			p, err := profile.Load(hash)
			if err != nil || len(p.Runs) == 0 {
				continue
			}

			var total int64
			for _, run := range p.Runs {
				total += run.DurationMS
			}
			avg := total / int64(len(p.Runs))

			lastRun := p.Runs[len(p.Runs)-1]
			sql := lastRun.NormalizedSQL
			if len(sql) > 60 {
				sql = sql[:57] + "..."
			}

			summaries = append(summaries, profileSummary{
				Hash:          hash,
				NormalizedSQL: sql,
				RunCount:      len(p.Runs),
				AvgDuration:   avg,
				LastRun:       lastRun.Timestamp.Format("Mon 02 Jan 15:04"),
				LastRunTime:   lastRun.Timestamp.Unix(),
				PlanHash:      lastRun.PlanHash,
			})
		}

		if len(summaries) == 0 {
			fmt.Fprintln(os.Stdout, "No profiled queries yet.")
			return nil
		}

		sort.Slice(summaries, func(i, j int) bool {
			return summaries[i].LastRunTime > summaries[j].LastRunTime
		})

		res := display.Result{
			Columns: []string{"Hash", "SQL", "Runs", "Avg (ms)", "Last Run", "Plan Hash"},
		}

		for _, s := range summaries {
			planHash := s.PlanHash
			if len(planHash) > 8 {
				planHash = planHash[:8]
			}
			hashShort := s.Hash
			if len(hashShort) > 8 {
				hashShort = hashShort[:8]
			}
			res.Rows = append(res.Rows, []string{
				hashShort,
				s.NormalizedSQL,
				fmt.Sprintf("%d", s.RunCount),
				fmt.Sprintf("%d", s.AvgDuration),
				s.LastRun,
				planHash,
			})
		}

		return display.Print(os.Stdout, res, display.FormatTable)
	},
}

var profileViewCmd = &cobra.Command{
	Use:   "view <hash>",
	Short: "Show detailed history for a specific query",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputHash := args[0]

		// Try exact match first
		p, err := profile.Load(inputHash)
		if err == nil && len(p.Runs) > 0 {
			return printProfileView(inputHash, p)
		}

		// Prefix match — scan directory
		dir := profile.ProfileDir()
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Profile not found for hash: %s\n", inputHash)
			return nil
		}

		var matches []string
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), inputHash) && strings.HasSuffix(entry.Name(), ".json") {
				matches = append(matches, strings.TrimSuffix(entry.Name(), ".json"))
			}
		}

		switch len(matches) {
		case 0:
			fmt.Fprintf(os.Stdout, "Profile not found for hash: %s\n", inputHash)
			return nil
		case 1:
			p, err := profile.Load(matches[0])
			if err != nil || len(p.Runs) == 0 {
				fmt.Fprintf(os.Stdout, "Profile not found for hash: %s\n", inputHash)
				return nil
			}
			return printProfileView(matches[0], p)
		default:
			fmt.Fprintf(os.Stdout, "Multiple profiles match '%s':\n", inputHash)
			for _, m := range matches {
				fmt.Fprintf(os.Stdout, "  %s\n", m[:8])
			}
			return nil
		}
	},
}

func printProfileView(hash string, p *profile.QueryProfile) error {
	var total int64
	var min, max int64
	if len(p.Runs) > 0 {
		min = p.Runs[0].DurationMS
		max = p.Runs[0].DurationMS
	}

	var durations []int64
	for _, run := range p.Runs {
		total += run.DurationMS
		if run.DurationMS < min {
			min = run.DurationMS
		}
		if run.DurationMS > max {
			max = run.DurationMS
		}
		durations = append(durations, run.DurationMS)
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	var median int64
	if len(durations) > 0 {
		mid := len(durations) / 2
		if len(durations)%2 == 0 {
			median = (durations[mid-1] + durations[mid]) / 2
		} else {
			median = durations[mid]
		}
	}

	avg := total / int64(len(p.Runs))

	fmt.Fprintf(os.Stdout, "Query Hash: %s\n", hash)
	fmt.Fprintf(os.Stdout, "SQL: %s\n", p.Runs[len(p.Runs)-1].NormalizedSQL)
	fmt.Fprintf(os.Stdout, "Runs: %d\n", len(p.Runs))
	fmt.Fprintf(os.Stdout, "Timing (ms) — Avg: %d | Median: %d | Min: %d | Max: %d\n\n", avg, median, min, max)

	res := display.Result{
		Columns: []string{"Timestamp", "Duration (ms)", "Rows", "Plan Hash"},
	}

	for _, run := range p.Runs {
		planHash := run.PlanHash
		if len(planHash) > 8 {
			planHash = planHash[:8]
		}
		res.Rows = append(res.Rows, []string{
			run.Timestamp.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%d", run.DurationMS),
			fmt.Sprintf("%d", run.RowsReturned),
			planHash,
		})
	}

	return display.Print(os.Stdout, res, display.FormatTable)
}

var profileClearCmd = &cobra.Command{
	Use:   "clear [hash]",
	Short: "Clear all profile data or a specific profile",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := profile.ProfileDir()

		if len(args) == 1 {
			hash := args[0]
			path := filepath.Join(dir, hash+".json")
			err := os.Remove(path)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintf(os.Stdout, "Profile not found for hash: %s\n", hash)
					return nil
				}
				return err
			}
			fmt.Fprintf(os.Stdout, "Cleared profile for hash: %s\n", hash)
			return nil
		}

		fmt.Fprint(os.Stdout, "Clear all query profiles? [y/N]: ")
		var resp string
		_, _ = fmt.Scanln(&resp)
		if resp != "y" && resp != "Y" && resp != "yes" && resp != "Yes" {
			fmt.Fprintln(os.Stdout, "Aborted.")
			return nil
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintln(os.Stdout, "Cleared 0 profile files.")
				return nil
			}
			return err
		}

		count := 0
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				if err := os.Remove(filepath.Join(dir, entry.Name())); err == nil {
					count++
				}
			}
		}

		fmt.Fprintf(os.Stdout, "Cleared %d profile files.\n", count)
		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileViewCmd)
	profileCmd.AddCommand(profileClearCmd)
}
