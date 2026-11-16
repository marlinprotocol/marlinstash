package cmd

import (
	"fmt"
	"marlinstash/db"
	"marlinstash/service"
	"marlinstash/types"
	"os"

	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const minRetentionDays = 3

func RunLogger(cmd *cobra.Command, args []string) error {
	// retention is off unless retention_days is set, values below the minimum are refused
	// as receiverTicketingClient reads up to the last day and a half of msg_recvs
	retentionDays := viper.GetInt("retention_days")
	if retentionDays != 0 && retentionDays < minRetentionDays {
		return fmt.Errorf("retention_days is %d, it has to be 0 (keep everything) or at least %d", retentionDays, minRetentionDays)
	}

	done := make(chan bool)
	dbWorker := db.CreateWorker(&pg.Options{
		Addr:     viper.GetString("database_host") + ":" + viper.GetString("database_port"),
		Database: viper.GetString("database_dbname"),
		User:     viper.GetString("database_username"),
		Password: viper.GetString("database_password"),
	}, retentionDays, done)
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
