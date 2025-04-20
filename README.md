# Kubernetes MCP Server 🚀

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![MCP](https://img.shields.io/badge/MCP-2025.03.26-purple?style=flat-square)](https://modelcontextprotocol.io/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Go Report Card](https://img.shields.io/badge/Go_Report-A+-success?style=flat-square&logo=go&logoColor=white)](https://github.com/briankscheong/k8s-mcp-server)
[![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=flat-square&logo=kubernetes&logoColor=white)](https://kubernetes.io/)

The Kubernetes MCP Server is a [Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) server that provides seamless integration with Kubernetes APIs, enabling advanced automation and interaction capabilities for developers, operators, and AI tools.

> **Note**: This project was inspired by and references the architecture of [GitHub's MCP Server](https://github.com/github/github-mcp-server). We acknowledge their excellent work which helped inform our implementation approach.

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
      "command": "k8smcp",
      "args": ["stdio"],
      "env": {
        "KUBECONFIG": "/path/to/your/kubeconfig"
      }
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
        "command": "k8smcp",
        "args": ["stdio"],
        "env": {
          "KUBECONFIG": "${input:kubeconfig_path}"
        }
      }
    }
  }
}
```

### Build from source

Clone the repository and build the binary:

```bash
git clone https://github.com/briankscheong/k8s-mcp-server.git
cd k8s-mcp-server
make build
```

Or install directly with Go:

```bash
go install github.com/briankscheong/k8s-mcp-server/cmd/k8s-mcp-server@latest
```

## Command Line Options ⌨️

```bash
Usage:
  k8smcp [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  http        Start HTTP server
  stdio       Start stdio server

Flags:
  -h, --help                     help for k8smcp
      --in-cluster               Use in-cluster config instead of kubeconfig file
      --kubeconfig string        Path to the kubeconfig file (default "/Users/briancheong/.kube/config")
      --log-commands             Log all commands and responses
      --log-file string          Path to log file (defaults to stderr)
      --namespace string         Default Kubernetes namespace (default "default")
      --read-only                Restrict operations to read-only (no create, update, delete) (default true)
      --resource-types strings   Comma separated list of Kubernetes resource types to enable (pods,deployments,services,configmaps,namespaces,nodes)
  -v, --version                  version for k8smcp
```

## Server Transport Options 🔄

### stdio (Stable)

The `stdio` transport is the default and most stable option, recommended for most users:

```bash
k8smcp stdio --kubeconfig=/path/to/your/kubeconfig
```

### SSE (Not Available Yet)

The `sse` transport option is under active development and will provide support for HTTP-based JSON-RPC message (de)serialization.

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

- **scale_deployment** - Scale a deployment to a specific number of replicas
  - `namespace`: Deployment namespace (string, optional, defaults to current namespace)
  - `name`: Deployment name (string, required)
  - `replicas`: Number of replicas (number, required)

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

## Future Enhancements 🔮

- Support for HTTP SSE Transport layer
- Dynamic tool creation via OpenAPI schema
- Enhanced RBAC integration for fine-grained access control
- Support for custom resource definitions (CRDs)
- Helm chart management capabilities
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
