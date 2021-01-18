package DB

import (
	"database/sql/driver"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/marlinprotocol/PersistentLogs/types"
	log "github.com/sirupsen/logrus"
)

// Database connection structure
type DBConnection struct {
	Host     string
	Port     string
	DBName   string
	Username string
	Password string
	conn     *driver.Driver

	DBEntriesTodo chan []types.DBEntryLine
	DBEntriesDone chan types.DBEntryLine
}

// ---------------------- DB CONNECT INTERFACE --------------------------------

// Run acts as the entry point to DB side connector for DB interface.
// DB Connector requires two channels for getting lines to add to DB / sending lines to PeriodicMonitor to notify it that insert was success
func (d *DBConnection) Run() {
	log.Info("Starting DB Connect Handler")
	for {
		err := d.createDBConnection()

		if err != nil {
			log.Error("Error encountered while creating DB Connection: ", err)
			os.Exit(1)
		}

		// ----- ROSHAN: write a while loop which waits for an entry from DBEntriesTodo chan,
		// adds it to database and then pushes the same struct onto DBEntriesDone for
		// periodic monitor to register onto the disk. If DB error, goto REATTEMPT_CONNECTION
		// will have to use reflection for playing well with select on []types.DBEntryLine

		// REATTEMPT_CONNECTION:
		log.Info("Error encountered with connection to the DB. Attempting reconnect post 1 second.")
		time.Sleep(1 * time.Second)
	}
}

func (d *DBConnection) createDBConnection() error {
	// ---- ROSHAN: create a DB connection and return error if any
	return nil
}
