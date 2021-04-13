package cmd

import (
	"marlinstash/db"
	"marlinstash/service"
	"marlinstash/types"
	"os"

	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RunLogger(cmd *cobra.Command, args []string) error {
	done := make(chan bool)
	dbWorker := db.CreateWorker(&pg.Options{
		Addr:     viper.GetString("database_host") + ":" + viper.GetString("database_port"),
		Database: viper.GetString("database_dbname"),
		User:     viper.GetString("database_username"),
		Password: viper.GetString("database_password"),
	}, done)
	go dbWorker.Run()

	// TODO: Use done, closed when db routine is ending
	// Shouldn't happen in normal operation, but handle nevertheless

	var services []types.Service
	err := viper.UnmarshalKey("services", &services)

	if err != nil {
		log.Error("Error while retrieving service list from config file.")
		os.Exit(1)
	}

	for _, srv := range services {
		go service.Run(srv, dbWorker.Entries, dbWorker.InodeOffsetReqs, dbWorker.ResetOffsetReqs)
	}

	infChan := make(chan struct{})
	select {
	case <-infChan:
		os.Exit(1)
	}

	return nil
}
