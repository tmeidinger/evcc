package cmd

import (
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// meterCmd represents the meter command
var meterCmd = &cobra.Command{
	Use:   "meter [name]",
	Short: "Query configured meters",
	Run:   runMeter,
}

func init() {
	rootCmd.AddCommand(meterCmd)
}

func runMeter(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config
	conf := loadConfigFile(cfgFile)

	// setup mqtt
	if viper.Get("mqtt") != nil {
		provider.MQTT = provider.NewMqttClient(conf.Mqtt.Broker, conf.Mqtt.User, conf.Mqtt.Password, clientID(), 1)
	}

	cp := &ConfigProvider{}
	cp.configureMeters(conf)

	meters := cp.meters
	if len(args) == 1 {
		arg := args[0]
		meters = map[string]api.Meter{arg: cp.Meter(arg)}
	}

	for name, v := range meters {
		if len(meters) != 1 {
			fmt.Println(name)
		}

		dumpAPIs(v)
	}
}
