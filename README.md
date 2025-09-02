# zopen-mcp-server

`zopen-mcp-server` is a Go-based server that provides a remote interface to the `zopen` command-line tool, allowing users to manage z/OS packages through the Model Context Protocol (MCP).

## Features

- **Remote Execution**: Run `zopen` commands on a remote z/OS system via SSH.
- **Local Execution**: Run `zopen` commands on the local machine.
- **MCP Integration**: Exposes `zopen` functionality as a set of tools that can be used by any MCP-compatible client.

## Security Model
By default, zopen-mcp-server communicates over stdio (standard input/output). When launched by a parent application (like Crush), this creates a direct and isolated communication channel. This method is inherently secure because the server is not exposed to a network port, preventing any unauthorized external connections.

When running in remote mode, the server uses SSH to execute commands on the target z/OS system. All actions are performed with the permissions of the SSH user provided. It is crucial to use an SSH key with the appropriate level of authority for the tasks you intend to perform.

## Prerequisites

- Go 1.23 or later
- An environment with `zopen` installed (either locally or on a remote z/OS system)

## Build and Run

A `Makefile` is provided to simplify the build and run process.

### Build

To build the server, run:

```sh
make build
```

This will create an executable named `zopen-mcp-server` in the project directory.

### Run

To run the server, use the `run` target:

```sh
make run
```

By default, the server runs in **local mode**. To run in **remote mode**, you can pass command-line flags:

```sh
./zopen-mcp-server --remote --host <your-zos-host> --user <your-user> --key <path-to-ssh-key>
```

### Clean

To clean up the build artifacts, run:

```sh
make clean
```

## Available Tools

The following `zopen` commands are available as tools:

- `zopen_list`: Lists information about zopen community packages.
- `zopen_query`: Lists local or remote info about zopen community packages.
- `zopen_install`: Installs one or more zopen community packages.
- `zopen_remove`: Removes installed zopen community packages.
- `zopen_upgrade`: Upgrades existing zopen community packages.
- `zopen_info`: Displays detailed information about a package.
- `zopen_version`: Displays the installed zopen version.
