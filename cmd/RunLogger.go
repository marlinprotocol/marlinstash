package cmd

import (
	"github.com/marlinprotocol/PersistentLogs/modules/DB"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RunLogger(cmd *cobra.Command, args []string) error {
	// Channel creation
	// var todoChans []types.DBEntryLine

	_ = DB.DBConnection{
		Host:     viper.GetString("database_host"),
		Port:     viper.GetString("database_port"),
		DBName:   viper.GetString("database_dbname"),
		Username: viper.GetString("database_username"),
		Password: viper.GetString("database_password"),
	}

	return nil
}
