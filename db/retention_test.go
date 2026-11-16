package db

import (
	"context"
	"fmt"
	"marlinstash/types"
	"os"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
)

// These tests need a Postgres database and are skipped without one. Run with
//
//	MARLINSTASH_TEST_DATABASE_URL=postgres://user:password@127.0.0.1:5432/dbname?sslmode=disable go test ./db/
//
// Everything is created in a schema of its own, which is dropped at the end.
func testOptions(t *testing.T) *pg.Options {
	t.Helper()
	url := os.Getenv("MARLINSTASH_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("MARLINSTASH_TEST_DATABASE_URL is not set")
	}
	opt, err := pg.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}

	schema := fmt.Sprintf("marlinstash_test_%d", time.Now().UnixNano())
	admin := pg.Connect(opt)
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		admin.Close()
	})

	opt.OnConnect = func(ctx context.Context, cn *pg.Conn) error {
		_, err := cn.Exec("SET search_path = " + schema)
		return err
	}
	return opt
}

func probeLine(ts time.Time, msgid int) string {
	return fmt.Sprintf("[%s] [info] [probe] Msg log: %d, cluster: 0xabc, relay: 127.0.0.1:1", ts.Format("2006-01-02 15:04:05.000"), msgid)
}

func count(t *testing.T, db *pg.DB, table string) int {
	t.Helper()
	var n int
	if _, err := db.QueryOne(pg.Scan(&n), "SELECT count(*) FROM "+table); err != nil {
		t.Fatal(err)
	}
	return n
}

func hasIndex(t *testing.T, db *pg.DB, name string) bool {
	t.Helper()
	var n int
	if _, err := db.QueryOne(pg.Scan(&n), "SELECT count(*) FROM pg_indexes WHERE schemaname = current_schema() AND indexname = ?", name); err != nil {
		t.Fatal(err)
	}
	return n == 1
}

func hasColumn(t *testing.T, db *pg.DB, table string, column string) bool {
	t.Helper()
	var n int
	if _, err := db.QueryOne(pg.Scan(&n), "SELECT count(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?", table, column); err != nil {
		t.Fatal(err)
	}
	return n == 1
}

func TestTsIndexMigration(t *testing.T) {
	opt := testOptions(t)
	db := pg.Connect(opt)
	defer db.Close()

	w := CreateWorker(opt, 0, make(chan bool))
	if err := w.setup(db); err != nil {
		t.Fatal(err)
	}
	if !hasIndex(t, db, "msg_recvs_ts_idx") || !hasIndex(t, db, "msg_sends_ts_idx") {
		t.Fatal("ts index missing on a new database")
	}

	// database as left by the version before the index was added
	if _, err := db.Exec("DROP INDEX msg_recvs_ts_idx; DROP INDEX msg_sends_ts_idx; UPDATE migration_states SET version = 2"); err != nil {
		t.Fatal(err)
	}
	if err := w.setup(db); err != nil {
		t.Fatal(err)
	}
	if !hasIndex(t, db, "msg_recvs_ts_idx") || !hasIndex(t, db, "msg_sends_ts_idx") {
		t.Fatal("ts index missing after upgrade")
	}
	// setup runs on every reconnect
	if err := w.setup(db); err != nil {
		t.Fatal(err)
	}

	// retention is off: nothing else changes and nothing is deleted
	if hasColumn(t, db, "entry_lines", "stored_at") {
		t.Fatal("stored_at added with retention off")
	}
	old := time.Now().Add(-400 * 24 * time.Hour)
	if err := w.processEntry(db, &types.EntryLine{Service: "probe", Host: "h", Inode: 1, Offset: 1, Message: probeLine(old, 1)}); err != nil {
		t.Fatal(err)
	}
	if err := w.applyRetention(db, time.Now()); err != nil {
		t.Fatal(err)
	}
	if count(t, db, "msg_recvs") != 1 || count(t, db, "entry_lines") != 1 {
		t.Fatal("rows deleted with retention off")
	}
}

func TestRetention(t *testing.T) {
	opt := testOptions(t)
	db := pg.Connect(opt)
	defer db.Close()

	// database that was in use before retention was enabled
	w := CreateWorker(opt, 0, make(chan bool))
	if err := w.setup(db); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	day := 24 * time.Hour
	insert := func(inode uint64, offset uint64, ts time.Time) {
		t.Helper()
		if err := w.processEntry(db, &types.EntryLine{Service: "probe", Host: "h", Inode: inode, Offset: offset, Message: probeLine(ts, int(offset))}); err != nil {
			t.Fatal(err)
		}
	}
	insert(1, 1, now.Add(-30*day))
	insert(1, 2, now.Add(-15*day))
	insert(1, 3, now.Add(-13*day))
	insert(1, 4, now.Add(-time.Hour))

	w = CreateWorker(opt, 14, make(chan bool))
	if err := w.setup(db); err != nil {
		t.Fatal(err)
	}
	if err := w.setup(db); err != nil {
		t.Fatal(err)
	}
	if !hasColumn(t, db, "entry_lines", "stored_at") || !hasColumn(t, db, "archived_entry_lines", "stored_at") {
		t.Fatal("stored_at missing with retention on")
	}

	// lines stored after retention is enabled
	insert(2, 1, now.Add(-30*day))
	insert(2, 2, now.Add(-time.Minute))
	// one of the old raw lines was stored 20 days ago
	if _, err := db.Exec("UPDATE entry_lines SET stored_at = ? WHERE inode = 1 AND \"offset\" = 1", now.Add(-20*day)); err != nil {
		t.Fatal(err)
	}

	// archival still works with the extra column, done the way the worker does it on a reset
	if _, err := db.Exec("INSERT INTO archived_entry_lines SELECT * FROM entry_lines WHERE inode = 2; DELETE FROM entry_lines WHERE inode = 2"); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.pipelines["probe"].Archive(tx, "probe", "h", 2); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if count(t, db, "archived_msg_recvs") != 2 || count(t, db, "archived_entry_lines") != 2 {
		t.Fatal("archive did not move the rows")
	}

	if err := w.applyRetention(db, now); err != nil {
		t.Fatal(err)
	}

	// msg_recvs is cut on the log time: 30 and 15 days old are gone, 13 days and 1 hour stay
	var offsets []int
	if _, err := db.Query(&offsets, "SELECT \"offset\" FROM msg_recvs ORDER BY \"offset\""); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(offsets) != "[3 4]" {
		t.Fatalf("msg_recvs offsets left: %v, want [3 4]", offsets)
	}
	// archived_msg_recvs the same way: the 30 days old row is gone
	if n := count(t, db, "archived_msg_recvs"); n != 1 {
		t.Fatalf("archived_msg_recvs rows left: %d, want 1", n)
	}
	// raw lines are cut on the time they were stored: only the one stored 20 days ago is gone
	if n := count(t, db, "entry_lines"); n != 3 {
		t.Fatalf("entry_lines rows left: %d, want 3", n)
	}
	if n := count(t, db, "archived_entry_lines"); n != 2 {
		t.Fatalf("archived_entry_lines rows left: %d, want 2", n)
	}

	// 15 days later everything stored today is past retention
	if err := w.applyRetention(db, now.Add(15*day)); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"msg_recvs", "archived_msg_recvs", "entry_lines", "archived_entry_lines"} {
		if n := count(t, db, table); n != 0 {
			t.Fatalf("%s rows left: %d, want 0", table, n)
		}
	}
}
