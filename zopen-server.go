// zopen_server.go
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
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Configuration ---
// Config holds the server's startup configuration, parsed from command-line flags.
type Config struct {
	Remote bool
	Host   string
	User   string
	Key    string
	Port   int
}

// --- Command Execution Logic ---

// ZopenExecutor handles the logic of running zopen commands, either locally or via SSH.
type ZopenExecutor struct {
	config *Config
}

// NewZopenExecutor creates a new executor based on the server's configuration.
func NewZopenExecutor(config *Config) *ZopenExecutor {
	if config.Remote {
		log.Printf("Executor initialized in REMOTE mode for host: %s", config.Host)
	} else {
		log.Println("Executor initialized in LOCAL mode.")
	}
	return &ZopenExecutor{config: config}
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

	log.Printf("Executing: %s", strings.Join(commandToRun, " "))
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

// --- Tool Definitions ---

// ZopenTools holds the server configuration and defines the tool methods.
type ZopenTools struct {
	Config *Config
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

// --- Main Server ---

func main() {
	config := &Config{}
	flag.BoolVar(&config.Remote, "remote", false, "Run in remote mode. Requires SSH details.")
	flag.StringVar(&config.Host, "host", "", "Remote z/OS hostname or IP (required for remote mode)")
	flag.StringVar(&config.User, "user", "", "SSH username for the remote system")
	flag.StringVar(&config.Key, "key", "", "Path to the SSH private key file")
	flag.IntVar(&config.Port, "port", 22, "SSH port number (default: 22)")
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

	mode := "LOCAL"
	if config.Remote {
		mode = "REMOTE"
	}
	log.Printf("Starting Zopen MCP server in %s mode...", mode)

	ctx := context.Background()
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server exited with error: %v", err)
	}
}
