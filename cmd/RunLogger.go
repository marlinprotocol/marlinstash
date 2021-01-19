package cmd

import (
	"marlinstash/db"
	"marlinstash/service"
	"marlinstash/types"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RunLogger(cmd *cobra.Command, args []string) error {

	dbWorker := db.CreateWorker(&db.Config{
		Host:     viper.GetString("database_host"),
		Port:     viper.GetString("database_port"),
		DBName:   viper.GetString("database_dbname"),
		User:     viper.GetString("database_username"),
		Password: viper.GetString("database_password"),
	})

	var services []types.Service
	err := viper.UnmarshalKey("services", &services)

	if err != nil {
		log.Error("Error while retrieving service list from config file.")
		os.Exit(1)
	}

	for _, srv := range services {
		go service.Run(srv, dbWorker.Entries)
	}

	infChan := make(chan struct{})
	select {
	case <-infChan:
		os.Exit(1)
	}

	return nil
}
