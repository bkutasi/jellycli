/*
 * Jellycli is a terminal music player for Jellyfin.
 * Copyright (C) 2020 Tero Vierimaa
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path"
	// "io" // Removed as MultiWriter is not used
	"strings"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"tryffel.net/go/jellycli/api"
	"tryffel.net/go/jellycli/api/jellyfin"
	"tryffel.net/go/jellycli/config"
	"tryffel.net/go/jellycli/player"
	"tryffel.net/go/jellycli/task"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Long: `Jellycli is a terminal music player for Jellyfin servers.

`,

	Run: func(cmd *cobra.Command, args []string) {
		initConfig() // Keep this for initial config loading and file creation
		_, err := initApplication()
		if err != nil {
			logrus.Fatalf("Failed to initialize application: %v", err)
		}
		// The application logic (run, stop) is now handled within initApplication
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file")
}

func initConfig() {
	// default config dir is ~/.config/jellycli
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configDir, err := os.UserConfigDir()
		if err != nil {
			logrus.Errorf("cannot determine config directory: %v", err)
			configDir = ""
		} else {
			configDir = path.Join(configDir, "jellycli")
		}

		viper.AddConfigPath(configDir)
		viper.SetConfigFile(path.Join(configDir, "jellycli.yaml"))
	}

	// env variables
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvPrefix("jellycli")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = config.NewConfigFile(cfgFile)
			if err != nil {
				logrus.Fatalf("create config file: %v", err)
			}
		} else {
			logrus.Fatalf("read config file: %v", err)
		}
	}

	// create new config file, save empty config file.
	err := config.ConfigFromViper()
	if err != nil {
		logrus.Fatalf("read config file: %v", err)
	}

	err = config.SaveConfig()
	if err != nil {
		logrus.Fatalf("save config file: %v", err)
	}

	file := viper.ConfigFileUsed()
	config.ConfigFile = file
}

// initLogging configures logrus to output only to Stderr.
func initLogging() error {
	level, err := logrus.ParseLevel(config.AppConfig.Player.LogLevel)
	if err != nil {
		// Log directly to stderr if parsing fails, before SetOutput is called
		fmt.Fprintf(os.Stderr, "Error parsing log level '%s': %v. Defaulting to INFO.\n", config.AppConfig.Player.LogLevel, err)
		level = logrus.InfoLevel // Default to Info level if parsing fails
	}

	logrus.SetLevel(level)
	format := &prefixed.TextFormatter{
		ForceColors:      true, // Enable colors for terminal output
		DisableColors:    false,
		ForceFormatting:  true,
		DisableTimestamp: false,
		DisableUppercase: false,
		FullTimestamp:    true,
		TimestampFormat:  "15:04:05.000",
		DisableSorting:   false,
		QuoteEmptyFields: false,
		QuoteCharacter:   "'",
		SpacePadding:     0,
		Once:             sync.Once{},
	}
	logrus.SetFormatter(format)

	// Set output directly to stderr
	logrus.SetOutput(os.Stderr)
	config.LogFile = "" // Indicate no log file is used

	// Log confirmation message *after* setting output
	logrus.Infof("Logging initialized to Stderr at level: %s", level.String())
	return nil // No file descriptor to return
}

// --- Application Lifecycle Logic ---

type app struct {
	server      api.MediaServer
	player      *player.Player
	// logfile     *os.File // Removed, logging goes to Stderr
}

func initApplication() (*app, error) {
	// Initialize logging (outputs only to Stderr)
	err := initLogging()
	if err != nil {
		// Error should have been logged within initLogging
		return nil, fmt.Errorf("init logging: %w", err)
	}

	a := &app{}
	// Log output is set to Stderr by initLogging

	logrus.Infof("############# %s v%s ############", config.AppName, config.Version)

	err = a.initServerConnection()
	if err != nil {
		logrus.Errorf("connect to server: %v", err) // Log error before returning
		return nil, fmt.Errorf("connect to server: %w", err)
	}

	err = a.initApp()
	if err != nil {
		logrus.Errorf("init application: %v", err) // Log error before returning
		return nil, fmt.Errorf("init application: %w", err)
	}

	// Save config *after* potential updates during server connection
	err = config.SaveConfig()
	if err != nil {
		// Log warning, but don't necessarily fail startup
		logrus.Warningf("save config file: %v", err)
	}

	a.run() // This now blocks until stop is called

	// stop() is called internally by run() via stopOnSignal()
	// We might need a way to return potential errors from stop() if necessary,
	// but for now, let's assume stop handles its errors internally via logging.
	// stopErr := a.stop()
	// if stopErr != nil {
	// 	logrus.Errorf("Error during application stop: %v", stopErr)
	// 	// Decide if this should be a fatal error or just logged
	// }

	// If initApplication completes without run() blocking forever (e.g., immediate signal),
	// Log file cleanup removed.

	return a, nil // Return the app instance, although it might have already stopped
}

func (a *app) initServerConnection() error {
	var err error
	serverType := strings.ToLower(config.AppConfig.Player.Server)
	logrus.Infof("Connecting to %s server...", serverType)
	switch serverType {
	case "jellyfin":
		a.server, err = jellyfin.NewJellyfin(&config.AppConfig.Jellyfin, &config.ViperStdConfigProvider{})
	default:
		return fmt.Errorf("unsupported backend: '%s'", config.AppConfig.Player.Server)
	}
	if err != nil {
		return fmt.Errorf("api init for %s: %w", serverType, err)
	}
	if err := a.server.ConnectionOk(); err != nil {
		return fmt.Errorf("no connection to %s server: %w", serverType, err)
	}
	logrus.Infof("Successfully connected to %s server.", serverType)

	// Update config with potentially refreshed credentials/settings from server
	conf := a.server.GetConfig()
	if config.AppConfig.Player.Server == "jellyfin" {
		jfConfig, ok := conf.(*config.Jellyfin)
		if ok {
			config.AppConfig.Jellyfin = *jfConfig
		}
	}
	return nil
}

func (a *app) initApp() error {
	var err error
	logrus.Info("Initializing player...")
	a.player, err = player.NewPlayer(a.server)
	if err != nil {
		return fmt.Errorf("create player: %w", err)
	}
	logrus.Info("Player initialized.")

	// MPRIS initialization removed.
	return nil
}

func (a *app) run() {
	if config.AppConfig.Player.EnableRemoteControl {
		remoteController, ok := a.server.(api.RemoteController)
		if ok {
			logrus.Info("Enabling remote control via server.")
			remoteController.SetPlayer(a.player)
			remoteController.SetQueue(a.player) // Assuming player implements QueueController
		} else {
			logrus.Warningf("Server type '%s' does not support remote control.", config.AppConfig.Player.Server)
		}
	}

	tasks := []task.Tasker{a.player, a.server}
	logrus.Info("Starting background tasks (player, server connection)...")
	for i, t := range tasks {
		taskName := fmt.Sprintf("task %d (%T)", i, t) // Get a basic name for logging
		err := t.Start()
		if err != nil {
			// Log fatal, as essential components failed to start
			logrus.Fatalf("Failed to start %s: %v", taskName, err)
			// Ensure stop is called for cleanup even on fatal startup error
			_ = a.stop() // Log errors within stop()
			os.Exit(1)   // Explicit exit after cleanup attempt
		}
		logrus.Debugf("Started %s.", taskName)
	}
	logrus.Info("Application started successfully. Running headless.")
	logrus.Info("Press Ctrl+C to exit.")

	// Block until signal is received
	a.stopOnSignal()

	// stop() is called by stopOnSignal, no need to call it again here.
	logrus.Info("Application run loop finished.")
}

func (a *app) stopOnSignal() {
	sigChan := catchSignals()
	sig := <-sigChan // Wait for signal
	logrus.Infof("Received signal: %s. Shutting down...", sig)
	err := a.stop()
	if err != nil {
		logrus.Errorf("Error during application stop triggered by signal: %v", err)
	} else {
		logrus.Info("Application stopped successfully.")
	}
	// No os.Exit here, let the main function handle exit.
}

func (a *app) stop() error {
	logrus.Info("Stopping application components...")
	// Stop tasks in reverse order? Player depends on server? Check dependencies.
	// Let's assume stopping player first is safer.
	tasks := []task.Tasker{a.player, a.server}
	var firstErr error

	// MPRIS related cleanup removed.


	for i := len(tasks) - 1; i >= 0; i-- { // Stop in reverse order of start
		t := tasks[i]
		taskName := fmt.Sprintf("task %d (%T)", i, t)
		logrus.Debugf("Stopping %s...", taskName)
		err := t.Stop()
		if err != nil {
			logrus.Errorf("Error stopping %s: %v", taskName, err)
			if firstErr == nil {
				firstErr = fmt.Errorf("error stopping %s: %w", taskName, err)
			}
		} else {
			logrus.Debugf("%s stopped.", taskName)
		}
	}
	// Log file closing logic removed.

	if firstErr != nil {
		logrus.Errorf("Completed stop sequence with errors.")
		return firstErr
	}

	logrus.Info("Application stop sequence completed.")
	return nil
}

func catchSignals() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGINT,  // Interrupt (Ctrl+C)
		syscall.SIGTERM) // Termination request
	logrus.Debug("Signal catcher initialized for SIGINT, SIGTERM.")
	return c
}
