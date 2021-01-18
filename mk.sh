#! /usr/bin/env bash

COLOR="\e[96m\e[1m"
ENDCOLOR="\e[0m"

echo -e "${COLOR}BUILDING persistentlogs with version $1 ${ENDCOLOR}"
export PERSISTENTLOGSBUILDVERSIONSTRING=$1
make release

echo -e "${COLOR}COPYING persistenctlogs to /usr/local/bin/ ${ENDCOLOR}"
make install

echo -e "${COLOR}COPYING configs to /etc/persistentlogs/config.yaml ${ENDCOLOR}"
cp example_config.yaml /etc/persistentlogs/config.yaml