package main

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/briankscheong/k8s-mcp-server/pkg/k8s"
	iolog "github.com/briankscheong/k8s-mcp-server/pkg/log"
	"github.com/briankscheong/k8s-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var version = "version"
var commit = "commit"
var date = "date"

var (
	rootCmd = &cobra.Command{
		Use:     "k8s-mcp",
		Short:   "Kubernetes MCP Server",
		Long:    `A Kubernetes MCP Server that provides tools for interacting with Kubernetes clusters.`,
		Version: fmt.Sprintf("Version: %s\nCommit: %s\nBuild Date: %s", version, commit, date),
	}

	stdioCmd = &cobra.Command{
		Use:   "stdio",
		Short: "Start stdio server",
		Long:  `Start a server that communicates via standard input/output streams using JSON-RPC messages.`,
		Run: func(_ *cobra.Command, _ []string) {
			logFile := viper.GetString("log-file")
			readOnly := viper.GetBool("read-only")
			kubeconfig := viper.GetString("kubeconfig")
			namespace := viper.GetString("namespace")
			inCluster := viper.GetBool("in-cluster")
			exportTranslations := viper.GetBool("export-translations")

			logger, err := initLogger(logFile)
			if err != nil {
				stdlog.Fatal("Failed to initialize logger:", err)
			}

			enabledToolsets := viper.GetStringSlice("toolsets")

			enabledResourceTypes := viper.GetStringSlice("resource-types")
			logCommands := viper.GetBool("log-commands")

			cfg := runConfig{
				readOnly:           readOnly,
				logger:             logger,
				logCommands:        logCommands,
				kubeconfig:         kubeconfig,
				namespace:          namespace,
				inCluster:          inCluster,
				exportTranslations: exportTranslations,
				enabledResources:   enabledResourceTypes,
				enabledToolsets:    enabledToolsets,
			}

			if err := runStdioServer(cfg); err != nil {
				stdlog.Fatal("Failed to run stdio server:", err)
			}
		},
	}

	httpCmd = &cobra.Command{
		Use:   "http",
		Short: "Start HTTP server",
		Long:  `Start a server that communicates via HTTP with Server-Sent Events (SSE).`,
		Run: func(_ *cobra.Command, _ []string) {
			// HTTP server implementation would go here
			stdlog.Fatal("HTTP server not yet implemented")
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.SetVersionTemplate("{{.Short}}\n{{.Version}}\n")

	// Add global flags for all commands
	rootCmd.PersistentFlags().StringSlice("resource-types", []string{},
		"Comma separated list of Kubernetes resource types to enable (pods,deployments,services,configmaps,namespaces,nodes)")
	rootCmd.PersistentFlags().Bool("read-only", true,
		"Restrict operations to read-only (no create, update, delete)")
	rootCmd.PersistentFlags().String("log-file", "",
		"Path to log file (defaults to stderr)")
	rootCmd.PersistentFlags().Bool("log-commands", false,
		"Log all commands and responses")
	rootCmd.PersistentFlags().String("namespace", "default",
		"Default Kubernetes namespace")

	// Kubernetes connection options
	defaultKubeconfig := ""
	if home := homedir.HomeDir(); home != "" {
		defaultKubeconfig = filepath.Join(home, ".kube", "config")
	}

	rootCmd.PersistentFlags().String("kubeconfig", defaultKubeconfig,
		"Path to the kubeconfig file")
	rootCmd.PersistentFlags().Bool("in-cluster", false,
		"Use in-cluster config instead of kubeconfig file")

	// Bind flags to viper
	_ = viper.BindPFlag("resource-types", rootCmd.PersistentFlags().Lookup("resource-types"))
	_ = viper.BindPFlag("read-only", rootCmd.PersistentFlags().Lookup("read-only"))
	_ = viper.BindPFlag("log-file", rootCmd.PersistentFlags().Lookup("log-file"))
	_ = viper.BindPFlag("log-commands", rootCmd.PersistentFlags().Lookup("log-commands"))
	_ = viper.BindPFlag("namespace", rootCmd.PersistentFlags().Lookup("namespace"))
	_ = viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	_ = viper.BindPFlag("in-cluster", rootCmd.PersistentFlags().Lookup("in-cluster"))

	// Add subcommands
	rootCmd.AddCommand(stdioCmd)
	rootCmd.AddCommand(httpCmd)
}

func initConfig() {
	// Enable environment variable binding
	viper.SetEnvPrefix("K8S_MCP")
	viper.AutomaticEnv()
}

func initLogger(outPath string) (*log.Logger, error) {
	logger := log.New()
	logger.SetLevel(log.InfoLevel)

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
	logger.SetFormatter(&log.JSONFormatter{})

	return logger, nil
}

type runConfig struct {
	readOnly           bool
	logger             *log.Logger
	logCommands        bool
	kubeconfig         string
	namespace          string
	inCluster          bool
	exportTranslations bool
	enabledResources   []string
	enabledToolsets    []string
}

func createK8sClient(cfg runConfig) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	if cfg.inCluster {
		// Use in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		// Use kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags("", cfg.kubeconfig)
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

func runStdioServer(cfg runConfig) error {
	// Create app context with signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create Kubernetes client
	k8sClient, err := createK8sClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Initialize translation helper
	t, dumpTranslations := translations.TranslationHelper()

	// Create client getter function
	getClient := func(_ context.Context) (*kubernetes.Clientset, error) {
		return k8sClient, nil
	}

	// Create MCP server
	k8sServer := k8s.NewServer(version)

	enabled := cfg.enabledToolsets

	// Create default toolsets
	k8s, err := k8s.InitToolsets(enabled, cfg.readOnly, getClient, t)
	if err != nil {
		stdlog.Fatal("Failed to initialize toolsets:", err)
	}

	// Register tools with the server
	k8s.RegisterTools(k8sServer)

	// Create STDIO server
	stdioServer := server.NewStdioServer(k8sServer)

	// Configure logger
	stdLogger := stdlog.New(cfg.logger.Writer(), "k8s-mcp-stdio: ", 0)
	stdioServer.SetErrorLogger(stdLogger)

	if cfg.exportTranslations {
		// Once server is initialized, all translations are loaded
		dumpTranslations()
	}

	// Start listening for messages
	errC := make(chan error, 1)
	go func() {
		in, out := io.Reader(os.Stdin), io.Writer(os.Stdout)

		if cfg.logCommands {
			loggedIO := iolog.NewIOLogger(in, out, cfg.logger)
			in, out = loggedIO, loggedIO
		}

		errC <- stdioServer.Listen(ctx, in, out)
	}()

	// Log startup message
	cfg.logger.Infof("Kubernetes MCP Server running on stdio")
	fmt.Fprintf(os.Stderr, "Kubernetes MCP Server running on stdio\n")

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		cfg.logger.Infof("Shutting down server...")
	case err := <-errC:
		if err != nil {
			return fmt.Errorf("error running server: %w", err)
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
