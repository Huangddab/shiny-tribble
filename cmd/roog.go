package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "data-server",
	Short: "Data Server",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("failed to bind flags: %w", err)
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once.
func Execute(context context.Context) error {
	// Set flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Initialize viper
	initViper()

	// Initialize logrus
	initLogrus()

	// // Initialize Database
	// database.InitializedDatabase(context)

	// // Initialize Broker
	// broker.InitializedClient(context)

	// Initialize all commands in explicit order
	initCommands()

	// Execute the root command
	return rootCmd.Execute()
}

// initCommands initializes all commands in a controlled order
func initCommands() {

	// Initialize user commands first
	initUserCommands()

	// Initialize bento commands next
	initBentoCommands()
}

// initViper initializes configuration
func initViper() {
	// Get config file from command line
	if cfgFile := viper.GetString("config"); cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			logrus.Errorf("failed to read config %s: %v", cfgFile, err)
			os.Exit(1)
		}
		logrus.Infof("using config file: %s", viper.ConfigFileUsed())

		// 即使指定了配置文件，也加载conf目录下的其他配置
		loadExtraConfig()
		return
	}

	// Set config name for the main app config
	viper.SetConfigName("app")
	viper.SetConfigType("yml")

	// Add search paths in order of priority
	viper.AddConfigPath("conf") // 配置目录
	viper.AddConfigPath(".")    // 当前目录

	// Executable file path
	execDir := filepath.Dir(os.Args[0])
	viper.AddConfigPath(filepath.Join(execDir, "conf"))
	viper.AddConfigPath(execDir)

	// 读取主配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logrus.Error("app.yml not found in conf/, ./, or exec-dir")
		} else {
			logrus.Errorf("failed to read app.yml: %v", err)
		}
		os.Exit(1)
	}

	logrus.Infof("using main config file: %s", viper.ConfigFileUsed())

	// 加载其他配置文件
	loadExtraConfig()
}

func loadExtraConfig() {
	// 获取主配置文件路径
	mainConfigPath := viper.ConfigFileUsed()
	if mainConfigPath == "" {
		logrus.Warn("no main config, skipping extra config")
		return
	}

	// 获取app.yml所在的目录
	configDir := filepath.Dir(mainConfigPath)

	// 检查目录是否存在
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		logrus.Warnf("config dir %s not exist, skip", configDir)
		return
	}

	// 仅获取所有.yml文件
	ymlFiles, err := filepath.Glob(filepath.Join(configDir, "*.yml"))
	if err != nil {
		logrus.Errorf("failed to glob yml files: %v", err)
		return
	}

	// 跳过已加载的主配置文件(app.yml)
	loadedCount := 0

	// 加载除了app.yml之外的所有yml文件
	for _, file := range ymlFiles {
		// 跳过主配置文件(app.yml)
		if file == mainConfigPath {
			continue
		}

		// 跳过非yml后缀的文件
		if filepath.Ext(file) != ".yml" {
			continue
		}

		v := viper.New()
		v.SetConfigFile(file)
		if err := v.ReadInConfig(); err != nil {
			logrus.Warnf("read config %s: %v", file, err)
			continue
		}

		// 将配置合并到主Viper实例
		if err := viper.MergeConfigMap(v.AllSettings()); err != nil {
			logrus.Warnf("merge config %s: %v", file, err)
			continue
		}

		logrus.Debugf("loaded extra config file: %s", file)
		loadedCount++
	}

	logrus.Infof("loaded %d extra configuration files from %s", loadedCount, configDir)
}

// initLogrus initializes the logger with configuration from viper
func initLogrus() {
	logLevel := viper.GetInt("log.level")
	if logLevel < int(logrus.PanicLevel) || logLevel > int(logrus.TraceLevel) {
		logLevel = int(logrus.InfoLevel)
	}
	logrus.SetLevel(logrus.Level(logLevel))
	logrus.SetReportCaller(viper.GetBool("log.show_caller"))
}
