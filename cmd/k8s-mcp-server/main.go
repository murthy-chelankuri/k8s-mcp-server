package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	stdlog "log"

	"github.com/briankscheong/k8s-mcp-server/pkg/k8s"
	iolog "github.com/briankscheong/k8s-mcp-server/pkg/log"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	logrus "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Config holds the common configuration for both transports
type Config struct {
	KubeConfig         string
	Namespace          string
	InCluster          bool
	ReadOnly           bool
	EnabledResources   []string
	EnabledToolsets    []string
	ExportTranslations bool
}

// StdioConfig holds STDIO specific configuration
type StdioConfig struct {
	Config
	LogFile     string
	LogCommands bool
}

// SSEConfig holds SSE specific configuration
type SSEConfig struct {
	Config
	Address string
}

var (
	rootCmd = &cobra.Command{
		Use:     "k8smcp",
		Short:   "Kubernetes MCP Server",
		Long:    `A Kubernetes MCP Server that provides tools for interacting with Kubernetes clusters.`,
		Version: fmt.Sprintf("Version: %s\nCommit: %s\nBuild Date: %s", version, commit, date),
	}

	stdioCmd = &cobra.Command{
		Use:   "stdio",
		Short: "Start stdio server",
		Long:  `Start a server that communicates via standard input/output streams using JSON-RPC messages.`,
		Run: func(_ *cobra.Command, _ []string) {
			cfg := StdioConfig{
				Config: Config{
					KubeConfig:         viper.GetString("kubeconfig"),
					Namespace:          viper.GetString("namespace"),
					InCluster:          viper.GetBool("in-cluster"),
					ReadOnly:           viper.GetBool("read-only"),
					EnabledResources:   viper.GetStringSlice("resource-types"),
					EnabledToolsets:    viper.GetStringSlice("toolsets"),
					ExportTranslations: viper.GetBool("export-translations"),
				},
				LogFile:     viper.GetString("log-file"),
				LogCommands: viper.GetBool("log-commands"),
			}

			if err := runStdioServer(cfg); err != nil {
				log.Fatal().Err(err).Msg("Failed to run stdio server")
			}
		},
	}

	sseCmd = &cobra.Command{
		Use:   "sse",
		Short: "Start HTTP SSE server",
		Long:  `Start a server that communicates via HTTP with Server-Sent Events (SSE).`,
		Run: func(_ *cobra.Command, _ []string) {
			cfg := SSEConfig{
				Config: Config{
					KubeConfig:         viper.GetString("kubeconfig"),
					Namespace:          viper.GetString("namespace"),
					InCluster:          viper.GetBool("in-cluster"),
					ReadOnly:           viper.GetBool("read-only"),
					EnabledResources:   viper.GetStringSlice("resource-types"),
					EnabledToolsets:    viper.GetStringSlice("toolsets"),
					ExportTranslations: viper.GetBool("export-translations"),
				},
				Address: viper.GetString("address"),
			}

			// Configure zerolog for SSE
			zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

			if err := runSSEServer(cfg); err != nil {
				log.Fatal().Err(err).Msg("Failed to run SSE server")
			}
		},
	}
)

func init() {
	// Kubernetes connection options
	defaultKubeconfig := ""
	if home := homedir.HomeDir(); home != "" {
		defaultKubeconfig = filepath.Join(home, ".kube", "config")
	}

	cobra.OnInitialize(initConfig)

	rootCmd.SetVersionTemplate("{{.Short}}\n{{.Version}}\n")

	// Add global flags for all commands
	rootCmd.PersistentFlags().StringSlice("resource-types", []string{},
		"Comma separated list of Kubernetes resource types to enable (pods,deployments,services,configmaps,namespaces,nodes)")
	rootCmd.PersistentFlags().Bool("read-only", true,
		"Restrict operations to read-only (no create, update, delete)")
	rootCmd.PersistentFlags().String("namespace", "default",
		"Default Kubernetes namespace")
	rootCmd.PersistentFlags().Bool("export-translations", false,
		"Save translations to a JSON file")
	rootCmd.PersistentFlags().StringSlice("toolsets", []string{"all"},
		"Comma separated list of tools to enable")
	rootCmd.PersistentFlags().String("kubeconfig", defaultKubeconfig,
		"Path to the kubeconfig file")
	rootCmd.PersistentFlags().Bool("in-cluster", false,
		"Use in-cluster config instead of kubeconfig file")

	// Add STDIO-specific flags
	stdioCmd.PersistentFlags().String("log-file", "",
		"Path to log file (defaults to stderr)")
	stdioCmd.PersistentFlags().Bool("log-commands", false,
		"Log all commands and responses")

	// Add SSE-specific flags
	sseCmd.PersistentFlags().String("address", getEnv("SSE_SERVER_ADDRESS", "8080"),
		"Address for SSE connections to be served")

	// Bind global flags to viper
	_ = viper.BindPFlag("resource-types", rootCmd.PersistentFlags().Lookup("resource-types"))
	_ = viper.BindPFlag("read-only", rootCmd.PersistentFlags().Lookup("read-only"))
	_ = viper.BindPFlag("namespace", rootCmd.PersistentFlags().Lookup("namespace"))
	_ = viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	_ = viper.BindPFlag("in-cluster", rootCmd.PersistentFlags().Lookup("in-cluster"))
	_ = viper.BindPFlag("export-translations", rootCmd.PersistentFlags().Lookup("export-translations"))
	_ = viper.BindPFlag("toolsets", rootCmd.PersistentFlags().Lookup("toolsets"))

	// Bind STDIO-specific flags to viper
	_ = viper.BindPFlag("log-file", stdioCmd.PersistentFlags().Lookup("log-file"))
	_ = viper.BindPFlag("log-commands", stdioCmd.PersistentFlags().Lookup("log-commands"))

	// Bind SSE-specific flags to viper
	_ = viper.BindPFlag("address", sseCmd.PersistentFlags().Lookup("address"))

	// Add subcommands
	rootCmd.AddCommand(stdioCmd)
	rootCmd.AddCommand(sseCmd)
}

func initConfig() {
	// Enable environment variable binding
	viper.SetEnvPrefix("K8S_MCP")
	viper.AutomaticEnv()
}

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

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func createK8sClient(kubeconfig string, inCluster bool) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	if inCluster {
		// Use in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		// Use kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return clientset, nil
}

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

	// Create default toolsets
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

func runStdioServer(cfg StdioConfig) error {
	// Create app context with signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize logger
	logger, err := initStdioLogger(cfg.LogFile)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Create Kubernetes client
	k8sClient, err := createK8sClient(cfg.KubeConfig, cfg.InCluster)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Initialize translation helper
	t, dumpTranslations := translations.TranslationHelper()

	// Create client getter function
	getClient := func(_ context.Context) (kubernetes.Interface, error) {
		return k8sClient, nil
	}

	// Create MCP server
	k8sServer := k8s.NewServer(version)

	enabled := cfg.EnabledToolsets

	// Create default toolsets
	k8s, err := k8s.InitToolsets(enabled, cfg.ReadOnly, getClient, t, cfg.EnabledResources)
	if err != nil {
		stdlog.Fatal("Failed to initialize toolsets:", err)
	}

	// Register tools with the server
	k8s.RegisterTools(k8sServer)

	// Create STDIO server
	stdioServer := server.NewStdioServer(k8sServer)

	// Configure logger
	stdLogger := stdlog.New(logger.Writer(), "k8s-mcp-stdio: ", 0)
	stdioServer.SetErrorLogger(stdLogger)

	if cfg.ExportTranslations {
		// Once server is initialized, all translations are loaded
		dumpTranslations()
	}

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

func runSSEServer(cfg SSEConfig) error {
	// Create app context with signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create MCP server
	k8sServer, err := setupK8sServer(cfg.Config)
	if err != nil {
		return err
	}

	// Create SSE server with options
	sseServer := server.NewSSEServer(k8sServer,
		server.WithBasePath("/mcp"),
		server.WithKeepAlive(true),
		server.WithSSEContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			// can add request-specific context values
			return ctx
		}),
	)

	// Create error channel
	errC := make(chan error, 1)

	address := fmt.Sprintf(":%s", cfg.Address)

	// Start the server in a goroutine
	go func() {
		log.Info().Str("address", address).Msg("Starting SSE server")
		errC <- sseServer.Start(address)
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
