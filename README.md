# zopen-mcp-server

`zopen-mcp-server` is a Go-based Model Context Protocol (MCP) server that provides AI assistants  with access to `zopen` and `zopen-generate` command-line tools for managing z/OS packages and porting open-source software to z/OS.

## Features

- **Remote Execution**: Run `zopen` commands on a remote z/OS system via SSH
- **Local Execution**: Run `zopen` and `zopen-generate` commands on your local machine
- **MCP Integration**: Exposes functionality as a set of tools that can be used by any MCP-compatible client
- **Project Generation**: Create zopen-compatible projects with customizable parameters
- **Metadata Discovery**: Query valid licenses, categories, and build systems
- **Build Support**: Build zopen projects with detailed output

## Security Model

By default, zopen-mcp-server communicates over stdio (standard input/output). When launched by a parent application, this creates a direct and isolated communication channel. This method is inherently secure because the server is not exposed to a network port, preventing any unauthorized external connections.

When running in remote mode, the server uses SSH to execute commands on the target z/OS system. All actions are performed with the permissions of the SSH user provided. It is crucial to use an SSH key with the appropriate level of authority for the tasks you intend to perform.

## Installation

### Option 1: Install with `go install` (Recommended)

```sh
go install github.com/zopencommunity/zopen-mcp-server@latest
```

This will install the `zopen-mcp-server` binary to your `$GOPATH/bin` directory (typically `~/go/bin`).

### Option 2: Build from Source

```sh
# Clone the repository
git clone https://github.com/zopencommunity/zopen-mcp-server.git
cd zopen-mcp-server

# Build using make
make build

# Or build directly with go
go build -o zopen-mcp-server zopen-server.go
```

## Prerequisites

- Go 1.23 or later
- An environment with `zopen` installed (either locally or on a remote z/OS system)
- For zopen-generate functionality: An environment with `zopen-generate` installed and accessible in the PATH

## Configuration


#### Local Mode (Default)

```json
{
  "mcpServers": {
    "zopen": {
      "command": "zopen-mcp-server",
      "args": []
    }
  }
}
```

If you installed with `go install`, make sure `~/go/bin` is in your PATH. Alternatively, use the full path:

```json
{
  "mcpServers": {
    "zopen": {
      "command": "/Users/yourname/go/bin/zopen-mcp-server",
      "args": []
    }
  }
}
```

#### Remote Mode (z/OS via SSH)

```json
{
  "mcpServers": {
    "zopen": {
      "command": "zopen-mcp-server",
      "args": [
        "--remote",
        "--host", "your-zos-hostname",
        "--user", "your-username",
        "--key", "/path/to/your/ssh/key"
      ]
    }
  }
}
```

#### With Debug Logging

```json
{
  "mcpServers": {
    "zopen": {
      "command": "zopen-mcp-server",
      "args": [],
      "env": {
        "DEBUG": "1"
      }
    }
  }
}
```

After updating the configuration, **restart the AI agent Desktop** for the changes to take effect.

## Usage

Once configured, the AI agent will have access to all zopen tools. You can ask it to:

- Port open-source software to z/OS
- Generate zopen project structures
- Build zopen projects
- Query package information
- Manage z/OS packages

See [AGENTS.md](AGENTS.md) for detailed instructions on how the ai agent should use these tools for porting software.

## Command Line Usage

You can also run the server directly from the command line:

### Local Mode

```sh
zopen-mcp-server
```

### Remote Mode

```sh
zopen-mcp-server --remote --host <zos-host> --user <username> --key <ssh-key-path>
```

### Available Flags

- `--remote`: Run in remote mode (requires SSH details)
- `--host`: Remote z/OS hostname or IP (required for remote mode)
- `--user`: SSH username for the remote system
- `--key`: Path to the SSH private key file
- `--port`: SSH port number (default: 22)
- `--zopen-path`: Path to the zopen executable (optional)

## Available Tools

The following `zopen` commands are available as tools:

- `zopen_list`: Lists information about zopen community packages.
- `zopen_query`: Lists local or remote info about zopen community packages.
- `zopen_install`: Installs one or more zopen community packages.
- `zopen_remove`: Removes installed zopen community packages.
- `zopen_upgrade`: Upgrades existing zopen community packages.
- `zopen_info`: Displays detailed information about a package.
- `zopen_version`: Displays the installed zopen version.
- `zopen_init`: Initializes the zopen environment.
- `zopen_clean`: Removes unused resources.
- `zopen_alt`: Switch between different versions of a package.
- `zopen_build`: Build a zopen project in the specified directory.

### zopen-generate Tools

The following `zopen-generate` commands are available as tools:

- `zopen_generate`: Generate a zopen compatible project with customizable parameters (including type and build_system).
- `zopen_generate_help`: Display help information for zopen-generate.
- `zopen_generate_version`: Display version information for zopen-generate.
- `zopen_generate_list_licenses`: List all valid license identifiers (returns JSON).
- `zopen_generate_list_categories`: List all valid project categories (returns JSON).
- `zopen_generate_list_build_systems`: List all valid build systems (returns JSON).
