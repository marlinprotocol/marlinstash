#! /usr/bin/env bash

COLOR="\e[96m\e[1m"
ENDCOLOR="\e[0m"

echo -e "${COLOR}BUILDING marlinstash with version $1 ${ENDCOLOR}"
export MARLINSTASHBUILDVERSIONSTRING=$1
make release

echo -e "${COLOR}COPYING marlinstash & marlinstash_migrate to /usr/local/bin/ ${ENDCOLOR}"
make install

mkdir -p /etc/marlinstash
if [ -f /etc/marlinstash/config.yml ]; then
	echo -e "${COLOR}KEEPING existing config at /etc/marlinstash/config.yml ${ENDCOLOR}"
else
	echo -e "${COLOR}COPYING configs to /etc/marlinstash/config.yml ${ENDCOLOR}"
	cp example_config.yml /etc/marlinstash/config.yml
	echo -e "${COLOR}EDIT /etc/marlinstash/config.yml before starting: database_* and logrootdir of the probe service ${ENDCOLOR}"
fi
