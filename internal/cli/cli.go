package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/adamroman0/cosyra-context/internal/cosyra"
	"github.com/adamroman0/cosyra-context/internal/hooks"
)

func Run(args []string, version string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return nil
	}

	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, version)
		return nil
	case "on":
		return runOn(args[1:], stdin, stdout)
	case "off":
		return runOff(args[1:], stdin, stdout)
	case "status":
		return runStatus(args[1:], stdout)
	case "hook":
		return runHook(args[1:], stdin, stdout)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runHook(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: cosyra hook <start|stop> --agent <agent>")
	}
	flags := flag.NewFlagSet("hook "+args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	agent := flags.String("agent", "", "agent name")
	projectFlag := flags.String("project", "", "project directory")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if *agent == "" {
		return errors.New("--agent is required")
	}
	project, err := cosyra.ResolveProjectRoot(*projectFlag)
	if err != nil {
		return err
	}

	switch args[0] {
	case "start":
		result, err := hooks.HandleStart(project, *agent, stdin)
		if err != nil {
			return err
		}
		if result.Context != "" {
			fmt.Fprint(stdout, result.Context)
		}
		return nil
	case "stop":
		_, err := hooks.HandleStop(project, *agent, stdin)
		return err
	default:
		return fmt.Errorf("unknown hook command %q", args[0])
	}
}

func runOn(args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet("on", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	yes := flags.Bool("yes", false, "accept prompts")
	projectFlag := flags.String("project", "", "project directory")
	toolsFlag := flags.String("tools", "", "comma-separated tools or all")
	all := flags.Bool("all", false, "enable all detected tools")
	if err := flags.Parse(args); err != nil {
		return err
	}

	project, err := cosyra.ResolveProjectRoot(*projectFlag)
	if err != nil {
		return err
	}
	detected := cosyra.DetectTools()

	fmt.Fprintf(stdout, "Cosyra will enable shared AI context for this project:\n\n  %s\n\n", project)
	if !*yes {
		if !confirm(stdin, stdout, "Enable Cosyra here? [y/N] ") {
			fmt.Fprintln(stdout, "Cosyra was not enabled.")
			return nil
		}
	}

	selected, err := selectTools(*toolsFlag, *all, *yes, detected, stdin, stdout)
	if err != nil {
		return err
	}

	cfg, err := cosyra.EnableProject(project, selected)
	if err != nil {
		return err
	}

	fmt.Fprintln(stdout, "Cosyra enabled for this project.")
	fmt.Fprintf(stdout, "\nProject:\n  %s\n", cfg.ProjectRoot)
	fmt.Fprintf(stdout, "\nContext:\n  %s\n", cfg.ContextPath())
	fmt.Fprintln(stdout, "\nConnected tools:")
	if len(cfg.EnabledTools) == 0 {
		fmt.Fprintln(stdout, "  none")
	} else {
		for _, tool := range cfg.EnabledTools {
			fmt.Fprintf(stdout, "  %s\n", cosyra.ToolDisplayName(tool))
		}
	}
	return nil
}

func runOff(args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet("off", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	yes := flags.Bool("yes", false, "accept prompts")
	purge := flags.Bool("purge", false, "remove .cosyra data")
	projectFlag := flags.String("project", "", "project directory")
	if err := flags.Parse(args); err != nil {
		return err
	}

	project, err := cosyra.ResolveProjectRoot(*projectFlag)
	if err != nil {
		return err
	}
	cfg, _ := cosyra.LoadConfig(project)

	fmt.Fprintf(stdout, "Cosyra will be disabled for this project:\n\n  %s\n\n", project)
	if cfg != nil && len(cfg.EnabledTools) > 0 {
		fmt.Fprintln(stdout, "Connected tools:")
		for _, tool := range cfg.EnabledTools {
			fmt.Fprintf(stdout, "  %s\n", cosyra.ToolDisplayName(tool))
		}
		fmt.Fprintln(stdout)
	}
	if !*yes {
		if !confirm(stdin, stdout, "Disable Cosyra here? [y/N] ") {
			fmt.Fprintln(stdout, "Cosyra was not disabled.")
			return nil
		}
	}

	if err := cosyra.DisableProject(project, *purge); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Cosyra disabled for this project.")
	if !*purge {
		fmt.Fprintln(stdout, "Local .cosyra data was left in place.")
	}
	return nil
}

func runStatus(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("status", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	projectFlag := flags.String("project", "", "project directory")
	if err := flags.Parse(args); err != nil {
		return err
	}

	project, err := cosyra.ResolveProjectRoot(*projectFlag)
	if err != nil {
		return err
	}
	cfg, err := cosyra.LoadConfig(project)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(stdout, "Cosyra is off for this project.")
			fmt.Fprintf(stdout, "\nProject:\n  %s\n", project)
			return nil
		}
		return err
	}

	fmt.Fprintln(stdout, "Cosyra is on for this project.")
	fmt.Fprintf(stdout, "\nProject:\n  %s\n", cfg.ProjectRoot)
	fmt.Fprintf(stdout, "\nContext:\n  %s\n", cfg.ContextPath())
	fmt.Fprintln(stdout, "\nTools:")
	for _, tool := range cosyra.AllTools() {
		state := "not connected"
		if cfg.ToolEnabled(tool) {
			state = "connected"
		}
		fmt.Fprintf(stdout, "  %-13s %s\n", cosyra.ToolDisplayName(tool), state)
	}
	return nil
}

func selectTools(toolsFlag string, all bool, yes bool, detected []cosyra.ToolDetection, stdin io.Reader, stdout io.Writer) ([]string, error) {
	if toolsFlag != "" {
		return parseTools(toolsFlag)
	}
	if all || yes {
		return detectedToolNames(detected), nil
	}

	fmt.Fprintln(stdout, "Select the AI tools to connect:")
	fmt.Fprintln(stdout, "  0) All detected tools")
	for i, tool := range detected {
		status := "not found"
		if tool.Detected {
			status = "detected"
		}
		fmt.Fprintf(stdout, "  %d) %-12s %s\n", i+1, cosyra.ToolDisplayName(tool.Name), status)
	}
	fmt.Fprint(stdout, "\nEnter selection [0]: ")

	var line string
	if _, err := fmt.Fscanln(stdin, &line); err != nil && line == "" {
		return detectedToolNames(detected), nil
	}
	line = strings.TrimSpace(line)
	if line == "" || line == "0" {
		return detectedToolNames(detected), nil
	}

	var selected []string
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var index int
		if _, err := fmt.Sscanf(part, "%d", &index); err == nil && index > 0 && index <= len(detected) {
			if detected[index-1].Detected {
				selected = append(selected, detected[index-1].Name)
			}
			continue
		}
		parsed, err := parseTools(part)
		if err != nil {
			return nil, err
		}
		selected = append(selected, parsed...)
	}
	return uniqueTools(selected), nil
}

func parseTools(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" || value == "all" {
		return cosyra.AllTools(), nil
	}
	var out []string
	for _, part := range strings.Split(value, ",") {
		tool := strings.ToLower(strings.TrimSpace(part))
		if !cosyra.ValidTool(tool) {
			return nil, fmt.Errorf("unknown tool %q", part)
		}
		out = append(out, tool)
	}
	return uniqueTools(out), nil
}

func detectedToolNames(detected []cosyra.ToolDetection) []string {
	var out []string
	for _, tool := range detected {
		if tool.Detected {
			out = append(out, tool.Name)
		}
	}
	return out
}

func uniqueTools(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range in {
		if seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func confirm(stdin io.Reader, stdout io.Writer, prompt string) bool {
	fmt.Fprint(stdout, prompt)
	var answer string
	if _, err := fmt.Fscanln(stdin, &answer); err != nil {
		return false
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cosyra <on|off|status|hook|version>")
}
