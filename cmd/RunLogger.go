package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"marlinstash/db"
)

func RunLogger(cmd *cobra.Command, args []string) error {
	// Channel creation
	// var todoChans []types.DBEntryLine

	_ = db.CreateWorker(&db.Config{
		Host:     viper.GetString("database_host"),
		Port:     viper.GetString("database_port"),
		DBName:   viper.GetString("database_dbname"),
		User:     viper.GetString("database_username"),
		Password: viper.GetString("database_password"),
	})

	return nil
}
