package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	stdlog "log"

	"github.com/briankscheong/k8s-mcp-server/pkg/k8s"
	iolog "github.com/briankscheong/k8s-mcp-server/pkg/log"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
	logrus "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Version information, populated during build
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Environment variable names - grouped by purpose
const (
	// Env prefix
	EnvPrefix = "K8S_MCP"

	// Kubernetes connection
	EnvKubeConfig = "KUBECONFIG"
	EnvNamespace  = "NAMESPACE"
	EnvInCluster  = "IN_CLUSTER"

	// Feature flags
	EnvReadOnly           = "READ_ONLY"
	EnvResourceTypes      = "RESOURCE_TYPES"
	EnvToolsets           = "TOOLSETS"
	EnvExportTranslations = "EXPORT_TRANSLATIONS"

	// stdio specific
	EnvLogFile     = "LOG_FILE"
	EnvLogCommands = "LOG_COMMANDS"

	// SSE specific
	EnvPort = "PORT"
)

// Config holds the common configuration for the server
type Config struct {
	// Kubernetes connection settings
	KubeConfig string `mapstructure:"kubeconfig"`
	Namespace  string `mapstructure:"namespace"`
	InCluster  bool   `mapstructure:"in-cluster"`

	// Feature flags
	ReadOnly           bool     `mapstructure:"read-only"`
	EnabledResources   []string `mapstructure:"resource-types"`
	EnabledToolsets    []string `mapstructure:"toolsets"`
	ExportTranslations bool     `mapstructure:"export-translations"`

	// Transport-specific config
	LogFile     string `mapstructure:"log-file"`
	LogCommands bool   `mapstructure:"log-commands"`
	Port        string `mapstructure:"port"`
}

// Validate checks that the configuration is valid
func (c *Config) Validate() error {
	// Validate required fields
	if c.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	// Validate that at least one resource type is enabled
	if len(c.EnabledResources) == 0 {
		return fmt.Errorf("at least one resource type must be enabled")
	}

	// Validate that at least one toolset is enabled
	if len(c.EnabledToolsets) == 0 {
		return fmt.Errorf("at least one toolset must be enabled")
	}

	// For SSE, validate the port
	if c.Port != "" {
		// Check if the port is a valid number
		if _, err := strconv.Atoi(c.Port); err != nil {
			return fmt.Errorf("invalid port number: %s", c.Port)
		}
	}

	return nil
}

var rootCmd = &cobra.Command{
	Use:     "k8smcp",
	Short:   "Kubernetes MCP Server",
	Long:    `A Kubernetes MCP Server that provides tools for interacting with Kubernetes clusters.`,
	Version: fmt.Sprintf("Version: %s\nCommit: %s\nBuild Date: %s", version, commit, date),
}

var stdioCmd = &cobra.Command{
	Use:   "stdio",
	Short: "Start stdio server",
	Long:  `Start a server that communicates via standard input/output streams using JSON-RPC messages.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Load the configuration
		var cfg Config
		if err := viper.Unmarshal(&cfg); err != nil {
			return fmt.Errorf("failed to parse server configuration: %w", err)
		}

		// Override with environment variables
		loadEnvOverrides(&cfg)

		// Validate the configuration
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid server configuration: %w", err)
		}

		return runStdioServer(cfg)
	},
}

var sseCmd = &cobra.Command{
	Use:   "sse",
	Short: "Start sse server",
	Long:  `Start a server that communicates via HTTP with Server-Sent Events (SSE).`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Load the configuration
		var cfg Config
		if err := viper.Unmarshal(&cfg); err != nil {
			return fmt.Errorf("failed to parse configuration: %w", err)
		}

		// Override with environment variables
		loadEnvOverrides(&cfg)

		// Validate the configuration
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}

		return runSSEServer(cfg)
	},
}

func init() {
	// Find default kubeconfig location
	defaultKubeconfig := ""
	if home := homedir.HomeDir(); home != "" {
		defaultKubeconfig = filepath.Join(home, ".kube", "config")
	}

	cobra.OnInitialize(initConfig)

	rootCmd.SetVersionTemplate("{{.Short}}\n{{.Version}}\n")
	rootCmd.SilenceUsage = false
	rootCmd.SilenceErrors = false

	// Add global flags for all commands
	rootCmd.PersistentFlags().StringSlice("resource-types", []string{"all"},
		"Comma separated list of Kubernetes resource types to enable (pods,deployments,services,configmaps,namespaces,nodes)")
	rootCmd.PersistentFlags().Bool("read-only", true,
		"Restrict operations to read-only (no create, update, delete)")
	rootCmd.PersistentFlags().String("namespace", "default",
		"Default Kubernetes namespace to target")
	rootCmd.PersistentFlags().Bool("export-translations", false,
		"Save translations to a JSON file")
	rootCmd.PersistentFlags().StringSlice("toolsets", []string{"all"},
		"Comma separated list of tools to enable")
	rootCmd.PersistentFlags().String("kubeconfig", defaultKubeconfig,
		"Path to the kubeconfig file")
	rootCmd.PersistentFlags().Bool("in-cluster", false,
		"Use in-cluster config instead of kubeconfig file")

	// Add stdio-specific flags
	stdioCmd.PersistentFlags().String("log-file", "",
		"Path to log file (defaults to stderr)")
	stdioCmd.PersistentFlags().Bool("log-commands", false,
		"Log all commands and responses")

	// Add SSE-specific flags
	sseCmd.PersistentFlags().String("port", "8080",
		"Port for SSE connections to be served")

	// Bind all flags to viper
	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		log.Fatal().Err(err).Msg("failed to bind root flags")
	}
	if err := viper.BindPFlags(stdioCmd.PersistentFlags()); err != nil {
		log.Fatal().Err(err).Msg("failed to bind stdio flags")
	}
	if err := viper.BindPFlags(sseCmd.PersistentFlags()); err != nil {
		log.Fatal().Err(err).Msg("failed to bind sse flags")
	}

	// Add subcommands
	rootCmd.AddCommand(stdioCmd)
	rootCmd.AddCommand(sseCmd)

	// Update command help with environment variable information
	addEnvHelpToCommand(rootCmd)
	addEnvHelpToCommand(stdioCmd)
	addEnvHelpToCommand(sseCmd)
}

// initConfig sets up viper for config handling
func initConfig() {
	// Enable environment variable binding
	viper.SetEnvPrefix(EnvPrefix)
	viper.AutomaticEnv()

	// Configure viper to use underscores in env vars
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
}

// loadEnvOverrides manually checks for environment variables and overrides config values
func loadEnvOverrides(cfg *Config) {
	// Check for kubernetes connection env vars
	if val, exists := os.LookupEnv(EnvPrefix + "_" + EnvKubeConfig); exists {
		cfg.KubeConfig = val
	}
	if val, exists := os.LookupEnv(EnvPrefix + "_" + EnvNamespace); exists {
		cfg.Namespace = val
	}
	if val, exists := os.LookupEnv(EnvPrefix + "_" + EnvInCluster); exists {
		cfg.InCluster = strings.ToLower(val) == "true" || val == "1"
	}

	// Check for feature flags
	if val, exists := os.LookupEnv(EnvPrefix + "_" + EnvReadOnly); exists {
		cfg.ReadOnly = strings.ToLower(val) == "true" || val == "1"
	}
	if val, exists := os.LookupEnv(EnvPrefix + "_" + EnvResourceTypes); exists && val != "" {
		cfg.EnabledResources = strings.Split(val, ",")
	}
	if val, exists := os.LookupEnv(EnvPrefix + "_" + EnvToolsets); exists && val != "" {
		cfg.EnabledToolsets = strings.Split(val, ",")
	}
	if val, exists := os.LookupEnv(EnvPrefix + "_" + EnvExportTranslations); exists {
		cfg.ExportTranslations = strings.ToLower(val) == "true" || val == "1"
	}

	// Check for transport-specific env vars
	if val, exists := os.LookupEnv(EnvPrefix + "_" + EnvLogFile); exists {
		cfg.LogFile = val
	}
	if val, exists := os.LookupEnv(EnvPrefix + "_" + EnvLogCommands); exists {
		cfg.LogCommands = strings.ToLower(val) == "true" || val == "1"
	}
	if val, exists := os.LookupEnv(EnvPrefix + "_" + EnvPort); exists {
		cfg.Port = val
	}
}

// addEnvHelpToCommand adds environment variable documentation to command help text
func addEnvHelpToCommand(cmd *cobra.Command) {
	originalHelp := cmd.Long

	// Define environment variables and their descriptions in slices
	var envVarNames []string
	var envVarDescs []string

	// Common env vars for all commands
	envVarNames = append(envVarNames,
		EnvKubeConfig,
		EnvNamespace,
		EnvInCluster,
		EnvReadOnly,
		EnvResourceTypes,
		EnvToolsets,
		EnvExportTranslations,
	)

	envVarDescs = append(envVarDescs,
		"Path to kubeconfig file",
		"Default Kubernetes namespace",
		"Use in-cluster config (true/false)",
		"Restrict to read-only operations (true/false)",
		"Comma-separated list of resource types",
		"Comma-separated list of toolsets to enable",
		"Export translations (true/false)",
	)

	// stdio specific env vars
	if cmd == stdioCmd {
		envVarNames = append(envVarNames,
			EnvLogFile,
			EnvLogCommands,
		)

		envVarDescs = append(envVarDescs,
			"Path to log file",
			"Log all commands (true/false)",
		)
	}

	// SSE specific env vars
	if cmd == sseCmd {
		envVarNames = append(envVarNames, EnvPort)
		envVarDescs = append(envVarDescs, "Port for SSE server")
	}

	// Calculate the maximum width needed for alignment
	maxWidth := 0
	for _, name := range envVarNames {
		fullName := fmt.Sprintf("%s_%s", EnvPrefix, name)
		if len(fullName) > maxWidth {
			maxWidth = len(fullName)
		}
	}

	// Create the help text with proper alignment
	envHelp := "\n\nEnvironment Variables:\n"

	// Add each relevant environment variable with proper alignment
	for i, name := range envVarNames {
		fullName := fmt.Sprintf("%s_%s", EnvPrefix, name)
		// Format with consistent padding for alignment
		envHelp += fmt.Sprintf("  %-*s   %s\n", maxWidth, fullName, envVarDescs[i])
	}

	cmd.Long = originalHelp + envHelp
}

// initStdioLogger creates and configures a logger for the stdio server
func initStdioLogger(outPath string) (*logrus.Logger, error) {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	if outPath == "" {
		// Default to stderr
		logger.SetOutput(os.Stderr)
		return logger, nil
	}

	file, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger.SetOutput(file)
	logger.SetFormatter(&logrus.JSONFormatter{})

	return logger, nil
}

// createK8sClient creates a Kubernetes clientset based on configuration
func createK8sClient(kubeconfig string, inCluster bool) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	var configSource string

	// First priority: explicitly set inCluster flag
	if inCluster {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
		configSource = "in-cluster (explicitly configured)"
	} else {
		// Second priority: valid kubeconfig file
		kubeconfigValid := false
		if kubeconfig != "" {
			if stat, statErr := os.Stat(kubeconfig); statErr == nil && stat.Size() > 0 {
				if cfg, cfgErr := clientcmd.BuildConfigFromFlags("", kubeconfig); cfgErr == nil {
					config = cfg
					kubeconfigValid = true
					configSource = fmt.Sprintf("kubeconfig file: %s", kubeconfig)
				}
			}
		}

		// Third priority: fallback to in-cluster if kubeconfig not valid
		if !kubeconfigValid {
			config, err = rest.InClusterConfig()
			if err != nil {
				// If all methods fail, provide a comprehensive error message
				return nil, fmt.Errorf("could not find valid authentication method: "+
					"kubeconfig file %q is invalid or missing and in-cluster config failed: %w",
					kubeconfig, err)
			}
			configSource = "in-cluster (fallback)"
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Log the config source for easier debugging
	log.Info().Str("source", configSource).Msg("Kubernetes client initialized")

	return clientset, nil
}

// setupK8sServer creates and configures the MCP server with K8s tools
func setupK8sServer(cfg Config) (*server.MCPServer, error) {
	// Create Kubernetes client
	k8sClient, err := createK8sClient(cfg.KubeConfig, cfg.InCluster)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Initialize translation helper
	t, dumpTranslations := translations.TranslationHelper()

	// Create client getter function
	getClient := func(_ context.Context) (kubernetes.Interface, error) {
		return k8sClient, nil
	}

	// Create MCP server
	k8sServer := k8s.NewServer(version)

	// Create toolsets
	k8sTools, err := k8s.InitToolsets(cfg.EnabledToolsets, cfg.ReadOnly, getClient, t, cfg.EnabledResources)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize toolsets: %w", err)
	}

	// Register tools with the server
	k8sTools.RegisterTools(k8sServer)

	// Export translations if requested
	if cfg.ExportTranslations {
		dumpTranslations()
	}

	return k8sServer, nil
}

// runStdioServer starts an MCP server using stdio transport
func runStdioServer(cfg Config) error {
	// Create app context with signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize logger
	logger, err := initStdioLogger(cfg.LogFile)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Create MCP server
	k8sServer, err := setupK8sServer(cfg)
	if err != nil {
		return err
	}

	// Create stdio server
	stdioServer := server.NewStdioServer(k8sServer)

	// Configure logger
	stdLogger := stdlog.New(logger.Writer(), "k8s-mcp-stdio: ", 0)
	stdioServer.SetErrorLogger(stdLogger)

	// Start listening for messages
	errC := make(chan error, 1)
	go func() {
		in, out := io.Reader(os.Stdin), io.Writer(os.Stdout)

		if cfg.LogCommands {
			loggedIO := iolog.NewIOLogger(in, out, logger)
			in, out = loggedIO, loggedIO
		}

		errC <- stdioServer.Listen(ctx, in, out)
	}()

	// Log startup message
	logger.Infof("Kubernetes MCP Server running on stdio")
	fmt.Fprintf(os.Stderr, "Kubernetes MCP Server running on stdio\n")

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		logger.Infof("Shutting down server...")
	case err := <-errC:
		if err != nil {
			return fmt.Errorf("error running server: %w", err)
		}
	}

	return nil
}

// runSSEServer starts an MCP server using SSE transport
func runSSEServer(cfg Config) error {
	// Create app context with signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create MCP server
	k8sServer, err := setupK8sServer(cfg)
	if err != nil {
		return err
	}

	// Create SSE server with options
	sseServer := server.NewSSEServer(k8sServer,
		server.WithBasePath("/mcp"),
		server.WithKeepAlive(true),
		server.WithSSEContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			// Can add request-specific context values
			return ctx
		}),
	)

	// Create error channel
	errC := make(chan error, 1)

	// Format port
	formattedPort := ":" + cfg.Port

	// Start the server in a goroutine
	go func() {
		log.Info().Str("port", cfg.Port).Msg("Starting SSE server")
		errC <- sseServer.Start(formattedPort)
	}()

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		log.Info().Msg("Shutting down server...")
		if err := sseServer.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Error during server shutdown")
		}
	case err := <-errC:
		if err != nil {
			log.Error().Err(err).Msg("Server error")
			if err := sseServer.Shutdown(ctx); err != nil {
				log.Error().Err(err).Msg("Error during server shutdown")
			}
			return err
		}
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
