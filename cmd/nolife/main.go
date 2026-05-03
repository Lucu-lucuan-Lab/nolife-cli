package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"nolife-cli/internal/app"
	"nolife-cli/internal/config"
	"nolife-cli/internal/ui"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

var (
	cfgFile    string
	outputDir  string
	episodes   string
	quality    string
	headless   bool
	noHeadless bool
	concurrent int
	skipExist  bool
)

var application *app.App

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "nolife",
	Short: "nolife - Download anime from various sites",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, configPath, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		application = app.New(cfg, configPath, headless, noHeadless)
		return nil
	},
}

var downloadCmd = &cobra.Command{
	Use:   "download [URL]",
	Short: "Download anime series or episodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		if application != nil {
			defer application.Close()
		}

		ctx, cancel := createInterruptContext()
		defer cancel()

		input := ""
		if len(args) > 0 {
			input = args[0]
		}

		opts := app.DownloadOptions{
			Input:        input,
			OutputDir:    outputDir,
			Episodes:     episodes,
			Quality:      quality,
			Concurrent:   concurrent,
			SkipExisting: skipExist,
		}

		return application.RunDownload(ctx, opts)
	},
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for anime by name",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if application != nil {
			defer application.Close()
		}

		ctx, cancel := createInterruptContext()
		defer cancel()

		query := args[0]
		return application.RunSearch(ctx, query)
	},
}

var listCmd = &cobra.Command{
	Use:   "list [URL]",
	Short: "List episodes from a series URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if application != nil {
			defer application.Close()
		}

		ctx, cancel := createInterruptContext()
		defer cancel()

		return application.RunList(ctx, args[0])
	},
}

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume interrupted downloads",
	RunE: func(cmd *cobra.Command, args []string) error {
		if application != nil {
			defer application.Close()
		}

		ctx, cancel := createInterruptContext()
		defer cancel()

		return application.RunResume(ctx)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.GetConfigPath()
		if err := config.SaveConfig(config.DefaultConfig(), path); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		ui.PrintSuccess("Configuration saved to: %s", path)
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintVersion(version, commit, date)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.PersistentFlags().BoolVar(&headless, "headless", true, "run browser headless")
	rootCmd.PersistentFlags().BoolVar(&noHeadless, "no-headless", false, "show browser window (for debugging)")

	downloadCmd.Flags().StringVarP(&outputDir, "output", "o", "", "output directory")
	downloadCmd.Flags().StringVarP(&episodes, "episodes", "e", "", "episode selection (e.g., '1-5', '1,3,7', 'all', 'latest')")
	downloadCmd.Flags().StringVarP(&quality, "quality", "q", "", "preferred quality (highest, 1080p, 720p, 480p)")
	downloadCmd.Flags().IntVar(&concurrent, "concurrent", 0, "concurrent downloads (0 = use config)")
	downloadCmd.Flags().BoolVar(&skipExist, "skip-existing", true, "skip existing files")

	configCmd.AddCommand(configInitCmd)

	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

func createInterruptContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		ui.PrintWarn("\nInterrupted. Saving progress & shutting down...")
		cancel()
	}()

	return ctx, cancel
}
