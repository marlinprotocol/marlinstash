/*
Copyright © 2020 MARLIN TEAM <info@marlin.pro>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/marlinprotocol/PersistentLogs/version"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

var cfgFile string
var logLevel string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "persistentlogs",
	Short:   "Persistent Logs saves logs from a file to a postgres instance",
	Long:    "Persistent Logs saves logs from a file to a postgres instance",
	Version: version.RootCmdVersion,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return readConfig()
	},
	RunE: RunLogger,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/persistentlogs/config.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func readConfig() error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigFile("/etc/persistentlogs/config.yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err == nil {
		var cfgVersionOnDisk = viper.GetInt("config_version")
		if cfgVersionOnDisk != version.CfgVersion {
			return errors.New("Cannot use the given config file as it does not match persistentlog's cfgversion. Wanted " + strconv.Itoa(version.CfgVersion) + " but found " + strconv.Itoa(cfgVersionOnDisk))
		}
	} else {
		log.Error("No config file available on local machine. Exiting")
		os.Exit(1)
	}
	return nil
}
