# Marlin Stash
Persist logs directly to a postgres instance

## Build and install
Build and install using
```
sudo su
sh mk.sh 0.0.1
```

## Configure
marlinstash reads `/etc/marlinstash/config.yml` by default, or the file given with `--config`. See `example_config.yml`:

- `database_*`: the Postgres database to write to. marlinstash creates its tables when it starts.
- `host`: name of this machine, stored with every record.
- `services`: log files to read. Each entry sets the directory (`logrootdir`) and a regex for the file names (`fileregex`). Files are checked for every 30 seconds.

The `probe` service stores every `Msg log` line of `eth_probe` in the `msg_recvs` table, which is read by [receiverTicketingClient](https://github.com/marlinprotocol/receiverTicketingClient).

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