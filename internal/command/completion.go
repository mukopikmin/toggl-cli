package command

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed completions/*
var completionScripts embed.FS

func (a App) completion(args []string) error {
	if len(args) != 1 {
		return usage("completion requires exactly one shell: bash, zsh, or fish")
	}
	switch args[0] {
	case "bash", "zsh", "fish":
		b, err := completionScripts.ReadFile("completions/" + args[0])
		if err != nil {
			return err
		}
		_, err = a.Out.Write(b)
		return err
	default:
		return usagef("unsupported completion shell: %s (expected bash, zsh, or fish)", args[0])
	}
}

// complete is the shell scripts' private, side-effect-free protocol. The last
// argument is the current (possibly empty) word; preceding words are complete.
// Never load configuration or call the API while completing.
func (a App) complete(words []string) error {
	for _, candidate := range completionCandidates(words) {
		if _, err := fmt.Fprintln(a.Out, candidate); err != nil {
			return err
		}
	}
	return nil
}

func completionCandidates(words []string) []string {
	current := ""
	if len(words) > 0 {
		current = words[len(words)-1]
		words = words[:len(words)-1]
	}
	filter := func(values, prefix string) []string {
		var matches []string
		for _, v := range strings.Fields(values) {
			v = prefix + v
			if strings.HasPrefix(v, current) {
				matches = append(matches, v)
			}
		}
		return matches
	}
	if len(words) == 0 {
		return filter("summary time-entry client project config init update completion --help -h --version", "")
	}
	command := words[0]
	args := words[1:]
	switch command {
	case "project", "client", "time-entry":
		verbs := "list"
		if command == "project" {
			verbs += " reorder sync"
		}
		if len(args) == 0 {
			return filter(verbs, "")
		}
		if args[0] != "list" {
			return nil
		}
		args = args[1:]
	case "completion":
		if len(args) == 0 {
			return filter("bash zsh fish", "")
		}
		return nil
	case "summary", "config", "update":
	default:
		return nil
	}

	options := "-f --format --watch"
	if command == "summary" {
		options += " -d --days --no-project --no-date --clipboard"
	}
	if command == "update" {
		options = "--channel"
	}
	values := func(option string) string {
		switch option {
		case "-f", "--format":
			return "csv tsv json table"
		case "--channel":
			return "stable nightly"
		}
		return ""
	}
	allowed := func(option string) bool {
		return strings.Contains(" "+options+" ", " "+option+" ")
	}
	pending := ""
	for _, arg := range args {
		if pending != "" {
			pending = ""
			continue
		}
		if arg == "--" {
			return nil
		}
		if strings.HasPrefix(arg, "-") && !allowed(arg) {
			if key, _, assigned := strings.Cut(arg, "="); assigned && allowed(key) {
				continue
			}
			if strings.HasPrefix(arg, "-f") && allowed("-f") || strings.HasPrefix(arg, "-d") && allowed("-d") {
				continue
			}
			return nil
		}
		if allowed(arg) && (values(arg) != "" || arg == "-d" || arg == "--days") {
			pending = arg
		}
	}
	if pending != "" {
		return filter(values(pending), "")
	}
	if key, _, ok := strings.Cut(current, "="); ok && allowed(key) {
		return filter(values(key), key+"=")
	}
	if strings.HasPrefix(current, "-f") && len(current) > 2 && allowed("-f") {
		return filter(values("-f"), "-f")
	}
	return filter(options, "")
}
