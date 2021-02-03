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

	// HIJACK DB STUB CODE - REMOVE LATER
	// h_entries := make(chan *types.EntryLine, 100)
	// h_inodeoffsetreq := make(chan *types.InodeOffsetReq, 100)

	// go func(h_e chan *types.EntryLine, h_i chan *types.InodeOffsetReq) {
	// 	for {
	// 		select {
	// 		case x := <-h_e:
	// 			log.Info("STUB ENTRY for message: ", x)
	// 		case x := <-h_i:
	// 			log.Info("STUB HIT for inode offset request", x)
	// 			x.Resp <- &types.InodeOffset{x.Service, x.Host, x.Inode, 0}
	// 		}
	// 	}
	// }(h_entries, h_inodeoffsetreq)

	// for _, srv := range services {
	// 	go service.Run(srv, h_entries, h_inodeoffsetreq)
	// }

	// HIJACKED CODE BELOW
	for _, srv := range services {
		go service.Run(srv, dbWorker.Entries, dbWorker.InodeOffsetReqs)
	}

	infChan := make(chan struct{})
	select {
	case <-infChan:
		os.Exit(1)
	}

	return nil
}
