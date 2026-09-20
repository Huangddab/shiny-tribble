package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"data-server/internal/simulator"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var simulatorCommand = &cobra.Command{
	Use:   "simulator",
	Short: "Run MQTT chemical reconnaissance device simulators",
	RunE:  runSimulatorCommand,
}

func initSimulatorCommand() {
	simulatorCommand.Flags().String("simulator.devices", "", "comma-separated device IDs")
	simulatorCommand.Flags().Int("simulator.count", 1, "number of generated simulator devices")
	simulatorCommand.Flags().Duration("simulator.interval", 2*time.Second, "Telemetry interval")
	simulatorCommand.Flags().Duration("simulator.alarm_every", 0, "start an alarm cycle at this interval; 0 disables alarm simulation")
	simulatorCommand.Flags().Duration("simulator.alarm_duration", 20*time.Second, "duration of each simulated alarm")
	simulatorCommand.Flags().String("simulator.pollutant", "DMMP", "simulated pollutant name")
	simulatorCommand.Flags().Float64("simulator.conc", 3.8, "simulated initial concentration in ppm")
	if err := viper.BindPFlags(simulatorCommand.Flags()); err != nil {
		logrus.Errorf("failed to bind simulator flags: %v", err)
	}
	rootCmd.AddCommand(simulatorCommand)
}

func runSimulatorCommand(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	config := simulator.Config{
		Broker:        viper.GetString("mqtt.broker"),
		ClientID:      viper.GetString("mqtt.client_id") + "-simulator",
		Username:      viper.GetString("mqtt.username"),
		Password:      viper.GetString("mqtt.password"),
		Devices:       simulator.DeviceIDs(viper.GetString("simulator.devices"), viper.GetInt("simulator.count")),
		Interval:      viper.GetDuration("simulator.interval"),
		AlarmEvery:    viper.GetDuration("simulator.alarm_every"),
		AlarmDuration: viper.GetDuration("simulator.alarm_duration"),
		Pollutant:     viper.GetString("simulator.pollutant"),
		PollutantConc: viper.GetFloat64("simulator.conc"),
	}
	if strings.TrimSpace(config.ClientID) == "-simulator" {
		config.ClientID = "data-server-simulator"
	}
	return simulator.Run(ctx, config)
}
