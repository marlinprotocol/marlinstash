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
- `services`: log files to read. Each entry sets the directory (`logrootdir`) and a regex for the file names (`fileregex`). Files are checked for every 30 seconds.

`logrootdir` in the example is a placeholder. Set it to the directory in which the probe writes its log (the directory `eth_probe` is started from when run with `nohup`). If the directory does not exist, marlinstash keeps running and logs `No files are read for this service` every 30 seconds till it does.

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