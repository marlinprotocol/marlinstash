#! /usr/bin/env bash

COLOR="\e[96m\e[1m"
ENDCOLOR="\e[0m"

echo -e "${COLOR}BUILDING marlinstash with version $1 ${ENDCOLOR}"
export MARLINSTASHBUILDVERSIONSTRING=$1
make release

echo -e "${COLOR}BUILDING marlinstash_migrate extra ${ENDCOLOR}"
make migrate

echo -e "${COLOR}COPYING marlinstash & marlinstash_migrate to /usr/local/bin/ ${ENDCOLOR}"
make install

echo -e "${COLOR}COPYING configs to /etc/marlinstash/config.yaml ${ENDCOLOR}"
mkdir -p /etc/marlinstash
cp example_config.yaml /etc/marlinstash/config.yaml
