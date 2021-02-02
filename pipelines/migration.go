package pipelines

import "github.com/go-pg/pg/v10"


type Migration struct {
	Forward, Backward func(db *pg.Tx) error
}

type MigrationState struct {
	Component string `pg:",notnull,unique"`
	Version   uint64 `pg:",notnull,use_zero"`
}

func ApplyMigrations(db *pg.DB, component string, migrations []*Migration, targetVersion uint64) error {
	migrationState := &MigrationState{component, 0}
	_, err := db.Model(migrationState).
		Where("component = ?", component).
		SelectOrInsert()

	if err != nil {
		return err
	}

	if migrationState.Version >= targetVersion {
		// Do nothing, already past target
		return nil
	}

	for _, migration := range migrations[migrationState.Version:] {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Close()

		err = migration.Forward(tx)
		if err != nil {
			tx.Rollback()
			return err
		}

		migrationState.Version++
		_, err = tx.Model(migrationState).
			OnConflict("(component) DO UPDATE").
			Set("version = EXCLUDED.version").
			Insert()
		if err != nil {
			tx.Rollback()
			return err
		}

		err = tx.Commit()
		if err != nil {
			return err
		}
	}

	return nil
}

