# Marlin Stash
Persist logs directly to a postgres instance

## Build and install
Build and install using
```
sudo su
sh mk.sh 0.0.1
```
This installs the binary to `/usr/local/bin/marlinstash` and, if there is no config yet, copies `example_config.yml` to `/etc/marlinstash/config.yml`. An existing config is not overwritten.

## Configure
marlinstash reads `/etc/marlinstash/config.yml` by default, or the file given with `--config`. See `example_config.yml`:

- `database_*`: the Postgres database to write to. marlinstash creates its tables when it starts.
- `host`: name of this machine, stored with every record.
- `retention_days`: optional, delete rows older than this many days. Off by default, see [Retention](#retention).
- `services`: log files to read. Each entry sets the directory (`logrootdir`) and a regex for the file names (`fileregex`). Files are checked for every 30 seconds.

`logrootdir` in the example is a placeholder. Set it to the directory in which the probe writes its log (the directory `eth_probe` is started from when run with `nohup`). If the directory does not exist, marlinstash keeps running and logs `No files are read for this service` every 30 seconds till it does.

The `probe` service stores every `Msg log` line of `eth_probe` in the `msg_recvs` table, which is read by [receiverTicketingClient](https://github.com/marlinprotocol/receiverTicketingClient).

## Retention
By default marlinstash never deletes anything. Every log line is stored in `entry_lines`, and every `Msg log` line again in `msg_recvs`, so the database keeps growing till the disk is full.

To delete old rows, set `retention_days` in the config and restart marlinstash:
```
retention_days: 14
```
One minute after start and every hour after that, marlinstash deletes

- rows of `msg_recvs`, `msg_sends` and their `archived_*` tables with a log time (`ts`) older than `retention_days` days.
- rows of `entry_lines` and `archived_entry_lines` that were stored more than `retention_days` days ago. These tables have no time column, so a `stored_at` column is added to them when retention is first enabled. Rows that are already there are counted from that moment, so they are deleted `retention_days` days after retention is enabled, not immediately.

Things to know before enabling it:

- receiverTicketingClient reads `msg_recvs` for the epochs after the last one it submitted tickets for (normally the last 1-2 days) and for the last day for telemetry. `retention_days` has to cover the longest time the ticketing client may be stopped or behind. If rows of epochs that are not submitted yet are deleted, the client skips those epochs and no tickets are issued for them. 14 days is a safe value, and values below 3 are refused at start.
- The first run on a large database deletes a lot of rows in one go. Nothing is stored while it runs, marlinstash continues from the same place in the log files after it is done.
- Postgres does not return the space of deleted rows to the disk. It is reused for new rows, so the database stops growing. Run `VACUUM FULL` on the tables to shrink them.
- Setting `retention_days` back to 0 stops the deletes. The `stored_at` column stays.

Independent of retention, `msg_recvs` and `msg_sends` get an index on `ts` when this version first starts. On a large table this takes a while, and nothing is stored till it is done.

## Run
Run using
```
marlinstash --config /etc/marlinstash/config.yml
```

## Version
Check version using
```
marlinstash -v
```

## Tests
```
go test ./service/ ./db/
```
The tests in `db` need a Postgres database and are skipped unless `MARLINSTASH_TEST_DATABASE_URL` is set, for example `postgres://user:password@127.0.0.1:5432/dbname?sslmode=disable`. They work in a schema of their own and drop it at the end.
