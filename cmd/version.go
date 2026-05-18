package cmd

import (
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		info := getBuildInfo()
		cmd.Printf("dbai %s\n", info.version)
		cmd.Printf("  Go version: %s\n", info.goVersion)
		cmd.Printf("  Platform: %s/%s\n", info.os, info.arch)
		if info.revision != "" {
			cmd.Printf("  Commit: %s\n", info.revision)
		}
		if info.buildTime != "" {
			cmd.Printf("  Built: %s\n", info.buildTime)
		}
		if info.dirty {
			cmd.Println("  Modified: yes")
		}
	},
}

type buildInfo struct {
	version   string
	goVersion string
	os        string
	arch      string
	revision  string
	buildTime string
	dirty     bool
}

func getBuildInfo() buildInfo {
	info := buildInfo{
		version:   "dev",
		goVersion: runtime.Version(),
		os:        runtime.GOOS,
		arch:      runtime.GOARCH,
	}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}

	if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		info.version = bi.Main.Version
	}

	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			info.revision = s.Value[:min(len(s.Value), 7)]
		case "vcs.time":
			info.buildTime = s.Value[:min(len(s.Value), 10)]
		case "vcs.modified":
			info.dirty = s.Value == "true"
		}
	}

	return info
}
