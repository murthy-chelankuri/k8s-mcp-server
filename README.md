# Kubernetes MCP Server 🚀

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![MCP](https://img.shields.io/badge/MCP-2025.03.26-purple?style=flat-square)](https://modelcontextprotocol.io/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=flat-square&logo=kubernetes&logoColor=white)](https://kubernetes.io/)

The Kubernetes MCP Server is a [Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) server that provides seamless integration with Kubernetes APIs, enabling advanced automation and interaction capabilities for developers, operators, and AI tools.

> **Note**: This project was inspired by and references the architecture of [GitHub's MCP Server](https://github.com/github/github-mcp-server). We acknowledge their excellent work which helped inform our implementation approach.

## Table of Contents

- [Kubernetes MCP Server 🚀](#kubernetes-mcp-server-)
  - [Table of Contents](#table-of-contents)
  - [Overview 📊](#overview-)
  - [Prerequisites ✅](#prerequisites-)
  - [Installation 💻](#installation-)
    - [Usage with Claude Desktop](#usage-with-claude-desktop)
    - [Usage with VS Code](#usage-with-vs-code)
    - [Usage with Cline](#usage-with-cline)
    - [Build from source](#build-from-source)
  - [Command Line Options ⌨️](#command-line-options-️)
  - [Server Transport Options 🔄](#server-transport-options-)
    - [stdio](#stdio)
    - [SSE](#sse)
  - [Access Control 🔒](#access-control-)
  - [Tools 🧰](#tools-)
    - [Resource Operations 📦](#resource-operations-)
    - [Management Operations ⚙️](#management-operations-️)
  - [Future Enhancements 🔮](#future-enhancements-)
  - [Contributing 👥](#contributing-)
  - [License ⚖️](#license-️)

## Overview 📊

This MCP server enables AI tools to interact with Kubernetes clusters using natural language, providing capabilities to:

- 🔍 Retrieve and analyze cluster resources
- 📈 Monitor deployments, pods, and services
- 🛠️ Execute common kubectl operations through AI interfaces
- 🔧 Troubleshoot cluster issues with AI assistance

## Prerequisites ✅

1. A Kubernetes cluster with API access
2. Valid kubeconfig file or service account credentials
3. Appropriate RBAC permissions for desired operations

## Installation 💻

### Usage with Claude Desktop

Add the following to your Claude Desktop configuration file (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS or `%APPDATA%\Claude\claude_desktop_config.json` on Windows):

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "path/to/k8smcp",
      "args": [
        "stdio",
        "--kubeconfig=/path/to/your/kubeconfig"
      ]
    }
  }
}
```

### Usage with VS Code

Add the following to your VS Code User Settings (JSON) file or `.vscode/mcp.json` in your workspace:

```json
{
  "mcp": {
    "inputs": [
      {
        "type": "promptString",
        "id": "kubeconfig_path",
        "description": "Path to kubeconfig file",
        "default": "${env:HOME}/.kube/config"
      }
    ],
    "servers": {
      "kubernetes": {
        "command": "path/to/k8smcp",
        "args": [
          "stdio",
          "--kubeconfig=/path/to/your/kubeconfig"
        ]
      }
    }
  }
}
```

### Usage with Cline

Add the following to your Cline configuration file (`path/to/cline_mcp_settings.json` after selecting "Configure MCP Servers"):

```json
{
  "mcpServers": {
    "kubernetes": {
      "disabled": false,
      "timeout": 60,
      "command": "path/to/k8smcp",
      "args": [
        "stdio",
        "--read-only=false"
        "--kubeconfig=/path/to/your/kubeconfig"
      ],
      "env": {
        "K8S_MCP_TOOLSETS": "all"
      },
      "transportType": "stdio"
    }
  }
}
```

Make sure to update the `command` value with the path to your k8smcp executable. You can set the server configurations either using `args` or `env`.

### Build from source

Clone the repository and build the binary:

```bash
git clone https://github.com/briankscheong/k8s-mcp-server.git
cd k8s-mcp-server
make build
```

## Command Line Options ⌨️

```txt
A Kubernetes MCP Server that provides tools for interacting with Kubernetes clusters.

Environment Variables:
  K8S_MCP_KUBECONFIG            Path to kubeconfig file
  K8S_MCP_NAMESPACE             Default Kubernetes namespace
  K8S_MCP_IN_CLUSTER            Use in-cluster config (true/false)
  K8S_MCP_READ_ONLY             Restrict to read-only operations (true/false)
  K8S_MCP_RESOURCE_TYPES        Comma-separated list of resource types
  K8S_MCP_TOOLSETS              Comma-separated list of toolsets to enable
  K8S_MCP_EXPORT_TRANSLATIONS   Export translations (true/false)

Usage:
  k8smcp [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  sse         Start sse server
  stdio       Start stdio server

Flags:
      --export-translations      Save translations to a JSON file
  -h, --help                     help for k8smcp
      --in-cluster               Use in-cluster config instead of kubeconfig file
      --kubeconfig string        Path to the kubeconfig file (default "/Users/briancheong/.kube/config")
      --namespace string         Default Kubernetes namespace to target (default "default")
      --read-only                Restrict operations to read-only (no create, update, delete) (default true)
      --resource-types strings   Comma separated list of Kubernetes resource types to enable (pods,deployments,services,configmaps,namespaces,nodes) (default [all])
      --toolsets strings         Comma separated list of tools to enable (default [all])
  -v, --version                  version for k8smcp

Use "k8smcp [command] --help" for more information about a command.
```

## Server Transport Options 🔄

### stdio

The `stdio` transport is the default and recommended option for most users for local integration:

```bash
k8smcp stdio --kubeconfig=/path/to/your/kubeconfig
```

### SSE

The `sse` transport provides support for HTTP-based JSON-RPC message transport. This can be helpful when deploying the server in a Kubernetes cluster that needs to expose a port for client connection.

```bash
k8smcp sse --in-cluster=true
```

> [!NOTE]
> The `--in-cluster=true` flag needs to be set if the server is deployed in a Kubernetes cluster.

## Access Control 🔒

By default, the server applies the permissions of the provided kubeconfig or service account. For enhanced security, you can:

1. Create a dedicated service account with restricted RBAC permissions
2. Set namespace limits to prevent cross-namespace operations
3. Enable read-only mode to prevent mutations to cluster state

## Tools 🧰

The Kubernetes MCP Server provides a comprehensive set of tools for interacting with your Kubernetes cluster.

### Resource Operations 📦

- **get_pod** - Get detailed information about a specific pod
  - `namespace`: Pod namespace (string, optional, defaults to current namespace)
  - `name`: Pod name (string, required)

- **list_pods** - List pods in a namespace
  - `namespace`: Namespace to list pods from (string, optional, defaults to current namespace)
  - `label_selector`: Filter pods by label selector (string, optional)
  - `field_selector`: Filter pods by field selector (string, optional)

- **get_pod_logs** - Get logs from a pod
  - `namespace`: Pod namespace (string, optional, defaults to current namespace)
  - `name`: Pod name (string, required)
  - `container`: Container name (string, optional, defaults to first container)
  - `tail_lines`: Number of lines to retrieve from the end (number, optional)
  - `previous`: Get logs from previous container instance (boolean, optional)

- **get_deployment** - Get information about a specific deployment
  - `namespace`: Deployment namespace (string, optional, defaults to current namespace)
  - `name`: Deployment name (string, required)

- **list_deployments** - List deployments in a namespace
  - `namespace`: Namespace to list deployments from (string, optional, defaults to current namespace)
  - `label_selector`: Filter deployments by label selector (string, optional)

- **get_service** - Get information about a specific service
  - `namespace`: Service namespace (string, optional, defaults to current namespace)
  - `name`: Service name (string, required)

- **list_services** - List services in a namespace
  - `namespace`: Namespace to list services from (string, optional, defaults to current namespace)
  - `label_selector`: Filter services by label selector (string, optional)

- **get_configmap** - Get information about a specific ConfigMap
  - `namespace`: ConfigMap namespace (string, optional, defaults to current namespace)
  - `name`: ConfigMap name (string, required)

- **list_configmaps** - List ConfigMaps in a namespace
  - `namespace`: Namespace to list ConfigMaps from (string, optional, defaults to current namespace)
  - `label_selector`: Filter ConfigMaps by label selector (string, optional)

- **list_namespaces** - List all namespaces in the cluster
  - No parameters required

- **list_nodes** - List all nodes in the cluster
  - No parameters required

### Management Operations ⚙️

- **delete_pod** - Delete a pod from a namespace
  - `namespace`: Pod namespace (string, optional, defaults to current namespace)
  - `name`: Pod name (string, required)
  - `grace_period_seconds`: Grace period before deletion (number, optional)

- **scale_deployment** - Scale a deployment to a specific number of replicas
  - `namespace`: Deployment namespace (string, optional, defaults to current namespace)
  - `name`: Deployment name (string, required)
  - `replicas`: Number of replicas (number, required)

> [!IMPORTANT]
> By default, tools that involve modification of resources in the cluster are disabled. To enable them, you have to set the `--read-only=false` flag or the `K8S_MCP_READ_ONLY=false` environment variable.

## Future Enhancements 🔮

- Enhanced RBAC integration for fine-grained access control
- Support for more kubernetes resources
- Support for custom resource definitions (CRDs)
- Helm chart management capabilities for deployment
- Cluster monitoring and alerting integration
- Support for multiple concurrent cluster connections

## Contributing 👥

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License ⚖️

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

---

<div align="center">
  
  ![Kubernetes + AI](https://img.shields.io/badge/Kubernetes%20%2B%20AI-The%20Future%20of%20DevOps-326CE5?style=for-the-badge&logo=kubernetes&logoColor=white)
  
  Built with ❤️ for the Kubernetes and AI communities.
</div>
