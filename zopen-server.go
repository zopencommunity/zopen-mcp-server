// zopen-server.go
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Configuration ---
// Config holds the server's startup configuration, parsed from command-line flags.
type Config struct {
	Remote    bool
	Host      string
	User      string
	Key       string
	Port      int
	ZopenPath string
}

// --- Command Execution Logic ---

// ZopenExecutor handles the logic of running zopen commands, either locally or via SSH.
type ZopenExecutor struct {
	config *Config
}

// ZopenGenerateExecutor handles the logic of running zopen-generate commands.
type ZopenGenerateExecutor struct {
	config *Config
}

// NewZopenExecutor creates a new executor based on the server's configuration.
func NewZopenExecutor(config *Config) *ZopenExecutor {
	// Logging disabled to avoid interfering with MCP stdio protocol
	return &ZopenExecutor{config: config}
}

// NewZopenGenerateExecutor creates a new executor based on the server's configuration.
func NewZopenGenerateExecutor(config *Config) *ZopenGenerateExecutor {
	// Logging disabled to avoid interfering with MCP stdio protocol
	return &ZopenGenerateExecutor{config: config}
}

// buildSSHCommand constructs the full SSH command for remote execution.
func (e *ZopenExecutor) buildSSHCommand(zopenArgs []string) []string {
	sshArgs := []string{"-p", fmt.Sprintf("%d", e.config.Port)}
	if e.config.Key != "" {
		sshArgs = append(sshArgs, "-i", e.config.Key)
	}
	sshArgs = append(sshArgs,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
	)

	target := e.config.Host
	if e.config.User != "" {
		target = fmt.Sprintf("%s@%s", e.config.User, e.config.Host)
	}
	sshArgs = append(sshArgs, target)

	// Quote arguments for the remote shell
	var quotedArgs []string
	for _, arg := range zopenArgs {
		quotedArgs = append(quotedArgs, fmt.Sprintf(`"%s"`, arg))
	}

	innerCommand := fmt.Sprintf(". ~/.profile && zopen %s", strings.Join(quotedArgs, " "))
	remoteCmd := fmt.Sprintf("/bin/sh -c \"%s\"", innerCommand)
	sshArgs = append(sshArgs, remoteCmd)

	return append([]string{"ssh"}, sshArgs...)
}

// RunCommand executes a zopen command either locally or remotely.
func (e *ZopenExecutor) RunCommand(ctx context.Context, zopenArgs []string) (string, error) {
	var commandToRun []string
	if e.config.Remote {
		commandToRun = e.buildSSHCommand(zopenArgs)
	} else {
		commandToRun = append([]string{"zopen"}, zopenArgs...)
	}

	// Log command execution if DEBUG is set
	if os.Getenv("DEBUG") != "" {
		log.Printf("Executing: %s", strings.Join(commandToRun, " "))
	}
	cmd := exec.CommandContext(ctx, commandToRun[0], commandToRun[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("❌ Error: Command '%s' not found. Is it in your PATH?", commandToRun[0])
		}
		return fmt.Sprintf("❌ Error (Exit Code: %d):\n%s", cmd.ProcessState.ExitCode(), stderr.String()), nil
	}

	output := stdout.String()
	if output == "" {
		return "✅ Command successful with no output.", nil
	}
	return output, nil
}

// RunCommand executes a zopen-generate command with the provided arguments.
func (e *ZopenGenerateExecutor) RunCommand(ctx context.Context, args []string) (string, error) {
	// Find zopen-generate in PATH
	commandPath, err := exec.LookPath("zopen-generate")
	if err != nil {
		return "", fmt.Errorf("❌ Error: zopen-generate not found in PATH")
	}

	// Log command execution if DEBUG is set
	if os.Getenv("DEBUG") != "" {
		log.Printf("Executing: %s %s", commandPath, strings.Join(args, " "))
	}
	cmd := exec.CommandContext(ctx, commandPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("❌ Error: Command '%s' not found", commandPath)
		}
		// Return stderr as part of the output, not as an error
		return fmt.Sprintf("❌ Error (Exit Code: %d):\n%s\n%s",
			cmd.ProcessState.ExitCode(),
			stderr.String(),
			stdout.String()), nil
	}

	output := stdout.String()
	if stderr.Len() > 0 {
		output = fmt.Sprintf("%s\n%s", output, stderr.String())
	}
	
	if output == "" {
		return "✅ Command successful with no output.", nil
	}
	return output, nil
}

// --- Tool Definitions ---

// ZopenTools holds the server configuration and defines the tool methods.
type ZopenTools struct {
	Config *Config
}

// --- ZopenGenerate Tool Definitions ---

// ZopenGenerateTools holds the server configuration and defines the zopen-generate tool methods.
type ZopenGenerateTools struct {
	Config *Config
}

// --- ZopenGenerate Tool ---
type ZopenGenerateParams struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Categories  string   `json:"categories"`
	License     string   `json:"license"`
	Type        string   `json:"type,omitempty"`
	BuildSystem string   `json:"build_system,omitempty"`
	StableUrl   string   `json:"stable_url,omitempty"`
	StableDeps  string   `json:"stable_deps,omitempty"`
	DevUrl      string   `json:"dev_url,omitempty"`
	DevDeps     string   `json:"dev_deps,omitempty"`
	BuildLine   string   `json:"build_line,omitempty"`
	RuntimeDeps string   `json:"runtime_deps,omitempty"`
	Force       bool     `json:"force,omitempty"`
}

func (t *ZopenGenerateTools) ZopenGenerate(ctx context.Context, req *mcp.CallToolRequest, args ZopenGenerateParams) (*mcp.CallToolResult, any, error) {
	// Validate required parameters
	if args.Name == "" || args.Description == "" || args.Categories == "" || args.License == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: "❌ Error: Required parameters missing. name, description, categories, and license are required.",
			}},
			IsError: true,
		}, nil, nil
	}

	// Build command arguments
	cmdArgs := []string{
		"--name", args.Name,
		"--description", args.Description,
		"--categories", args.Categories,
		"--license", args.License,
		"--non-interactive", // Always run in non-interactive mode
	}

	// Add optional parameters if provided
	if args.Type != "" {
		cmdArgs = append(cmdArgs, "--type", args.Type)
	}
	if args.BuildSystem != "" {
		cmdArgs = append(cmdArgs, "--build-system", args.BuildSystem)
	}
	if args.StableUrl != "" {
		cmdArgs = append(cmdArgs, "--stable-url", args.StableUrl)
	}
	if args.StableDeps != "" {
		cmdArgs = append(cmdArgs, "--stable-deps", args.StableDeps)
	}
	if args.DevUrl != "" {
		cmdArgs = append(cmdArgs, "--dev-url", args.DevUrl)
	}
	if args.DevDeps != "" {
		cmdArgs = append(cmdArgs, "--dev-deps", args.DevDeps)
	}
	if args.BuildLine != "" {
		cmdArgs = append(cmdArgs, "--build-line", args.BuildLine)
	}
	if args.RuntimeDeps != "" {
		cmdArgs = append(cmdArgs, "--runtime-deps", args.RuntimeDeps)
	}
	if args.Force {
		cmdArgs = append(cmdArgs, "--force")
	}

	executor := NewZopenGenerateExecutor(t.Config)
	output, err := executor.RunCommand(ctx, cmdArgs)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, nil, nil
	}

	isError := strings.HasPrefix(output, "❌")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
		IsError: isError,
	}, nil, nil
}

// --- ZopenGenerateHelp Tool ---
func (t *ZopenGenerateTools) ZopenGenerateHelp(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
	executor := NewZopenGenerateExecutor(t.Config)
	output, err := executor.RunCommand(ctx, []string{"--help"})
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
		IsError: false,
	}, nil, nil
}

// --- ZopenGenerateVersion Tool ---
func (t *ZopenGenerateTools) ZopenGenerateVersion(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
	executor := NewZopenGenerateExecutor(t.Config)
	output, err := executor.RunCommand(ctx, []string{"--version"})
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
		IsError: false,
	}, nil, nil
}

// Generic handler for zopen commands
func (t *ZopenTools) handleZopenCommand(ctx context.Context, zopenArgs []string) (*mcp.CallToolResult, any, error) {
	executor := NewZopenExecutor(t.Config)
	output, err := executor.RunCommand(ctx, zopenArgs)
	if err != nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, nil, nil
	}
	isError := strings.HasPrefix(output, "❌")
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: output}}, IsError: isError}, nil, nil
}

// --- ZopenList Tool ---
type ZopenListParams struct {
	Verbose bool `json:"verbose"`
}

func (t *ZopenTools) ZopenList(ctx context.Context, req *mcp.CallToolRequest, args ZopenListParams) (*mcp.CallToolResult, any, error) {
	zopenArgs := []string{"list"}
	if args.Verbose {
		zopenArgs = append(zopenArgs, "--verbose")
	}
	return t.handleZopenCommand(ctx, zopenArgs)
}

// --- ZopenQuery Tool ---
type ZopenQueryParams struct {
	Packages []string `json:"packages"`
	Verbose  bool     `json:"verbose"`
}

func (t *ZopenTools) ZopenQuery(ctx context.Context, req *mcp.CallToolRequest, args ZopenQueryParams) (*mcp.CallToolResult, any, error) {
	zopenArgs := []string{"query"}
	if args.Verbose {
		zopenArgs = append(zopenArgs, "--verbose")
	}
	if len(args.Packages) > 0 {
		zopenArgs = append(zopenArgs, args.Packages...)
	}
	return t.handleZopenCommand(ctx, zopenArgs)
}

// --- ZopenInstall Tool ---
type ZopenInstallParams struct {
	Packages []string `json:"packages"`
	Verbose  bool     `json:"verbose"`
}

func (t *ZopenTools) ZopenInstall(ctx context.Context, req *mcp.CallToolRequest, args ZopenInstallParams) (*mcp.CallToolResult, any, error) {
	zopenArgs := []string{"install"}
	if args.Verbose {
		zopenArgs = append(zopenArgs, "--verbose")
	}
	zopenArgs = append(zopenArgs, args.Packages...)
	return t.handleZopenCommand(ctx, zopenArgs)
}

// --- ZopenRemove Tool ---
type ZopenRemoveParams struct {
	Packages []string `json:"packages"`
	Verbose  bool     `json:"verbose"`
}

func (t *ZopenTools) ZopenRemove(ctx context.Context, req *mcp.CallToolRequest, args ZopenRemoveParams) (*mcp.CallToolResult, any, error) {
	zopenArgs := []string{"remove"}
	if args.Verbose {
		zopenArgs = append(zopenArgs, "--verbose")
	}
	zopenArgs = append(zopenArgs, args.Packages...)
	return t.handleZopenCommand(ctx, zopenArgs)
}

// --- ZopenUpgrade Tool ---
type ZopenUpgradeParams struct {
	Packages []string `json:"packages"`
	Verbose  bool     `json:"verbose"`
	Yes      bool     `json:"yes"`
}

func (t *ZopenTools) ZopenUpgrade(ctx context.Context, req *mcp.CallToolRequest, args ZopenUpgradeParams) (*mcp.CallToolResult, any, error) {
	zopenArgs := []string{"upgrade"}
	if args.Yes {
		zopenArgs = append(zopenArgs, "--yes")
	}
	if args.Verbose {
		zopenArgs = append(zopenArgs, "--verbose")
	}
	if len(args.Packages) > 0 {
		zopenArgs = append(zopenArgs, args.Packages...)
	}
	return t.handleZopenCommand(ctx, zopenArgs)
}

// --- ZopenInfo Tool ---
type ZopenInfoParams struct {
	Package string `json:"package"`
	Verbose bool   `json:"verbose"`
}

func (t *ZopenTools) ZopenInfo(ctx context.Context, req *mcp.CallToolRequest, args ZopenInfoParams) (*mcp.CallToolResult, any, error) {
	zopenArgs := []string{"info", args.Package}
	if args.Verbose {
		zopenArgs = append(zopenArgs, "--verbose")
	}
	return t.handleZopenCommand(ctx, zopenArgs)
}

// --- ZopenVersion Tool ---
func (t *ZopenTools) ZopenVersion(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
	return t.handleZopenCommand(ctx, []string{"version"})
}

// --- ZopenInit Tool ---
func (t *ZopenTools) ZopenInit(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
	return t.handleZopenCommand(ctx, []string{"init"})
}

// --- ZopenClean Tool ---
type ZopenCleanParams struct {
	Cache    bool `json:"cache"`
	Unused   bool `json:"unused"`
	Dangling bool `json:"dangling"`
	All      bool `json:"all"`
}

func (t *ZopenTools) ZopenClean(ctx context.Context, req *mcp.CallToolRequest, args ZopenCleanParams) (*mcp.CallToolResult, any, error) {
	zopenArgs := []string{"clean"}
	if args.Cache {
		zopenArgs = append(zopenArgs, "--cache")
	}
	if args.Unused {
		zopenArgs = append(zopenArgs, "--unused")
	}
	if args.Dangling {
		zopenArgs = append(zopenArgs, "--dangling")
	}
	if args.All {
		zopenArgs = append(zopenArgs, "--all")
	}
	return t.handleZopenCommand(ctx, zopenArgs)
}

// --- ZopenAlt Tool ---
type ZopenAltParams struct {
	Package string `json:"package"`
	Switch  string `json:"switch"`
}

func (t *ZopenTools) ZopenAlt(ctx context.Context, req *mcp.CallToolRequest, args ZopenAltParams) (*mcp.CallToolResult, any, error) {
	zopenArgs := []string{"alt"}
	if args.Package != "" {
		zopenArgs = append(zopenArgs, args.Package)
	}
	if args.Switch != "" {
		zopenArgs = append(zopenArgs, "-s", args.Switch)
	}
	return t.handleZopenCommand(ctx, zopenArgs)
}

// --- ZopenBuild Tool ---
type ZopenBuildParams struct {
	Directory string `json:"directory"`
	Verbose   bool   `json:"verbose"`
	Force     bool   `json:"force"`
}

func (t *ZopenTools) ZopenBuild(ctx context.Context, req *mcp.CallToolRequest, args ZopenBuildParams) (*mcp.CallToolResult, any, error) {
	if args.Directory == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: "❌ Error: directory parameter is required",
			}},
			IsError: true,
		}, nil, nil
	}

	// Build the command
	zopenArgs := []string{"build"}
	if args.Verbose {
		zopenArgs = append(zopenArgs, "-vv")
	}
	if args.Force {
		zopenArgs = append(zopenArgs, "-f")
	}

	// For local execution, we need to cd into the directory
	if !t.Config.Remote {
		// Get absolute path
		absPath, err := filepath.Abs(args.Directory)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: fmt.Sprintf("❌ Error: failed to get absolute path: %v", err),
				}},
				IsError: true,
			}, nil, nil
		}

		// Check if directory exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: fmt.Sprintf("❌ Error: directory does not exist: %s", absPath),
				}},
				IsError: true,
			}, nil, nil
		}

		// Execute in the directory
		cmd := exec.CommandContext(ctx, "zopen", zopenArgs...)
		cmd.Dir = absPath

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err = cmd.Run()
		output := stdout.String()
		if stderr.Len() > 0 {
			output = fmt.Sprintf("%s\n%s", output, stderr.String())
		}

		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: fmt.Sprintf("❌ Error (Exit Code: %d):\n%s", cmd.ProcessState.ExitCode(), output),
				}},
				IsError: true,
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: output}},
			IsError: false,
		}, nil, nil
	}

	// For remote execution, cd into directory before running zopen build
	return t.handleZopenCommandInDirectory(ctx, args.Directory, zopenArgs)
}

// handleZopenCommandInDirectory is similar to handleZopenCommand but changes directory first
func (t *ZopenTools) handleZopenCommandInDirectory(ctx context.Context, directory string, zopenArgs []string) (*mcp.CallToolResult, any, error) {
	executor := NewZopenExecutor(t.Config)

	if t.Config.Remote {
		// For remote, we need to modify the SSH command to cd first
		sshArgs := []string{"-p", fmt.Sprintf("%d", t.Config.Port)}
		if t.Config.Key != "" {
			sshArgs = append(sshArgs, "-i", t.Config.Key)
		}
		sshArgs = append(sshArgs,
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "LogLevel=ERROR",
		)

		target := t.Config.Host
		if t.Config.User != "" {
			target = fmt.Sprintf("%s@%s", t.Config.User, t.Config.Host)
		}
		sshArgs = append(sshArgs, target)

		// Quote arguments for the remote shell
		var quotedArgs []string
		for _, arg := range zopenArgs {
			quotedArgs = append(quotedArgs, fmt.Sprintf(`"%s"`, arg))
		}

		innerCommand := fmt.Sprintf(". ~/.profile && cd %s && zopen %s", directory, strings.Join(quotedArgs, " "))
		remoteCmd := fmt.Sprintf("/bin/sh -c \"%s\"", innerCommand)
		sshArgs = append(sshArgs, remoteCmd)

		commandToRun := append([]string{"ssh"}, sshArgs...)
		// Logging disabled for MCP stdio protocol
	// log.Printf("Executing: %s", strings.Join(commandToRun, " "))

		cmd := exec.CommandContext(ctx, commandToRun[0], commandToRun[1:]...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{
					Text: fmt.Sprintf("❌ Error (Exit Code: %d):\n%s", cmd.ProcessState.ExitCode(), stderr.String()),
				}},
				IsError: true,
			}, nil, nil
		}

		output := stdout.String()
		if output == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "✅ Command successful with no output."}},
				IsError: false,
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: output}},
			IsError: false,
		}, nil, nil
	}

	output, err := executor.RunCommand(ctx, zopenArgs)
	if err != nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, nil, nil
	}
	isError := strings.HasPrefix(output, "❌")
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: output}}, IsError: isError}, nil, nil
}

// --- ZopenBuildHelp Tool ---
func (t *ZopenTools) ZopenBuildHelp(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
	executor := NewZopenExecutor(t.Config)
	output, err := executor.RunCommand(ctx, []string{"build", "--help"})
	if err != nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, nil, nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: output}}, IsError: false}, nil, nil
}

// --- ZopenCreateRepo Tool ---
type ZopenCreateRepoParams struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	User        string `json:"user"`
}

func (t *ZopenTools) ZopenCreateRepo(ctx context.Context, req *mcp.CallToolRequest, args ZopenCreateRepoParams) (*mcp.CallToolResult, any, error) {
	if args.Name == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: "❌ Error: name parameter is required",
			}},
			IsError: true,
		}, nil, nil
	}

	// Build the command
	zopenArgs := []string{"create-repo", "-v", "-n", args.Name}

	if args.Description != "" {
		zopenArgs = append(zopenArgs, "-d", args.Description)
	}

	if args.User != "" {
		zopenArgs = append(zopenArgs, "-u", args.User)
	}

	executor := NewZopenExecutor(t.Config)
	output, err := executor.RunCommand(ctx, zopenArgs)
	if err != nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, nil, nil
	}
	isError := strings.HasPrefix(output, "❌")
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: output}}, IsError: isError}, nil, nil
}

// --- ZopenGenerateListLicenses Tool ---
func (t *ZopenGenerateTools) ZopenGenerateListLicenses(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
	executor := NewZopenGenerateExecutor(t.Config)
	output, err := executor.RunCommand(ctx, []string{"--json", "--list-licenses"})
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
		IsError: false,
	}, nil, nil
}

// --- ZopenGenerateListCategories Tool ---
func (t *ZopenGenerateTools) ZopenGenerateListCategories(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
	executor := NewZopenGenerateExecutor(t.Config)
	output, err := executor.RunCommand(ctx, []string{"--json", "--list-categories"})
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
		IsError: false,
	}, nil, nil
}

// --- ZopenGenerateListBuildSystems Tool ---
func (t *ZopenGenerateTools) ZopenGenerateListBuildSystems(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, any, error) {
	executor := NewZopenGenerateExecutor(t.Config)
	output, err := executor.RunCommand(ctx, []string{"--json", "--list-build-systems"})
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
		IsError: false,
	}, nil, nil
}

// --- Main Server ---

func main() {
	config := &Config{}
	flag.BoolVar(&config.Remote, "remote", false, "Run in remote mode. Requires SSH details.")
	flag.StringVar(&config.Host, "host", "", "Remote z/OS hostname or IP (required for remote mode)")
	flag.StringVar(&config.User, "user", "", "SSH username for the remote system")
	flag.StringVar(&config.Key, "key", "", "Path to the SSH private key file")
	flag.IntVar(&config.Port, "port", 22, "SSH port number (default: 22)")
	flag.StringVar(&config.ZopenPath, "zopen-path", "", "Path to the zopen executable (optional, will use PATH if not specified)")
	flag.Parse()

	if config.Remote && config.Host == "" {
		fmt.Println("Error: --host is required when using --remote mode.")
		flag.Usage()
		os.Exit(1)
	}


	server := mcp.NewServer(&mcp.Implementation{
		Name:    "Zopen Tools Server (Go)",
		Version: "1.0.0",
	}, nil)

	tools := &ZopenTools{Config: config}
	genTools := &ZopenGenerateTools{Config: config}

	// Register each tool individually
	mcp.AddTool(server, &mcp.Tool{Name: "zopen_list", Description: "Lists information about zopen community packages"}, tools.ZopenList)
	mcp.AddTool(server, &mcp.Tool{Name: "zopen_query", Description: "List local or remote info about zopen community packages"}, tools.ZopenQuery)
	mcp.AddTool(server, &mcp.Tool{Name: "zopen_install", Description: "Installs one or more zopen community packages"}, tools.ZopenInstall)
	mcp.AddTool(server, &mcp.Tool{Name: "zopen_remove", Description: "Removes installed zopen community packages"}, tools.ZopenRemove)
	mcp.AddTool(server, &mcp.Tool{Name: "zopen_upgrade", Description: "Upgrades existing zopen community packages"}, tools.ZopenUpgrade)
	mcp.AddTool(server, &mcp.Tool{Name: "zopen_info", Description: "Displays detailed information about a package"}, tools.ZopenInfo)
	mcp.AddTool(server, &mcp.Tool{Name: "zopen_version", Description: "Display the installed zopen version"}, tools.ZopenVersion)
	mcp.AddTool(server, &mcp.Tool{Name: "zopen_init", Description: "Initializes the zopen environment"}, tools.ZopenInit)
	mcp.AddTool(server, &mcp.Tool{Name: "zopen_clean", Description: "Removes unused resources"}, tools.ZopenClean)
	mcp.AddTool(server, &mcp.Tool{Name: "zopen_alt", Description: "Switch between different versions of a package"}, tools.ZopenAlt)
	mcp.AddTool(server, &mcp.Tool{Name: "zopen_build", Description: "Build a zopen project in the specified directory"}, tools.ZopenBuild)
	mcp.AddTool(server, &mcp.Tool{Name: "zopen_build_help", Description: "Display help information for zopen build"}, tools.ZopenBuildHelp)
	mcp.AddTool(server, &mcp.Tool{Name: "zopen_create_repo", Description: "Create a new port repository in zopencommunity (core contributors only)"}, tools.ZopenCreateRepo)

	// Register zopen-generate tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "zopen_generate",
		Description: "Generate a zopen compatible project with customizable parameters",
	}, genTools.ZopenGenerate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "zopen_generate_help",
		Description: "Display help information for zopen-generate",
	}, genTools.ZopenGenerateHelp)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "zopen_generate_version",
		Description: "Display version information for zopen-generate",
	}, genTools.ZopenGenerateVersion)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "zopen_generate_list_licenses",
		Description: "List all valid license identifiers (returns JSON)",
	}, genTools.ZopenGenerateListLicenses)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "zopen_generate_list_categories",
		Description: "List all valid project categories (returns JSON)",
	}, genTools.ZopenGenerateListCategories)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "zopen_generate_list_build_systems",
		Description: "List all valid build systems (returns JSON)",
	}, genTools.ZopenGenerateListBuildSystems)

	mode := "LOCAL"
	if config.Remote {
		mode = "REMOTE"
	}

	// Only log startup in debug mode (avoid interfering with MCP protocol)
	// MCP uses stdio for communication, so we minimize logging
	if os.Getenv("DEBUG") != "" {
		log.Printf("Starting Zopen MCP server in %s mode...", mode)
	}

	ctx := context.Background()
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		// Log to stderr is OK, but only in debug mode
		if os.Getenv("DEBUG") != "" {
			log.Fatalf("Server exited with error: %v", err)
		}
		os.Exit(1)
	}
}
