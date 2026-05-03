package ui

import (
	"fmt"
	"os"
)

func PrintSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stdout, PrefixOK+" "+SuccessStyle.Render(msg))
}

func PrintError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, PrefixFail+" "+ErrorStyle.Render(msg))
}

func PrintWarn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stdout, PrefixWarn+" "+WarningStyle.Render(msg))
}

func PrintInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stdout, PrefixInfo+" "+InfoStyle.Render(msg))
}

func PrintHeader(title string) {
	fmt.Println()
	fmt.Println(HeaderTagStyle.Render(title))
	fmt.Println()
}

type EpisodeDisplay struct {
	Number int
	Title  string
	Status string
}

func PrintEpisodeList(episodes []EpisodeDisplay) {
	PrintHeader("Episodes")
	for _, ep := range episodes {
		var status string
		switch ep.Status {
		case "completed":
			status = SuccessStyle.Render("✓")
		case "downloading":
			status = InfoStyle.Render("⟳")
		case "failed":
			status = ErrorStyle.Render("✗")
		case "pending":
			status = MutedStyle.Render("○")
		default:
			status = " "
		}
		fmt.Printf("  %s %s\n", status, ep.Title)
	}
	fmt.Println()
}

func PrintDownloadStatus(current, total int, title string) {
	counter := MutedStyle.Render(fmt.Sprintf("[%d/%d]", current, total))
	name := BoldStyle.Render(title)
	fmt.Printf("\n  %s %s\n", counter, name)
}

func PrintVersion(version, commit, date string) {
	title := TitleStyle.Render("nolife") + " " + MutedStyle.Render("— CLI")
	fmt.Println(title)
	fmt.Printf("  %s %s\n", MutedStyle.Render("Version:"), version)
	if commit != "" {
		fmt.Printf("  %s %s\n", MutedStyle.Render("Commit:"), commit)
	}
	if date != "" {
		fmt.Printf("  %s %s\n", MutedStyle.Render("Date:"), date)
	}
	fmt.Println()
}
