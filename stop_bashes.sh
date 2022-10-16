#!/bin/bash
SCRIPT=$(readlink -f "$0")
base_path=$(dirname "$SCRIPT")

# Env
if [ -f ${base_path}/.env ];then
  source ${base_path}/.env
else
  echo "ERROR!"
  echo "Bashes is not installed yet. Please run install.sh"
  exit
fi

pid=`cat ${base_path}/.pid`
kill -9 ${pid}
chrome_pid=`cat ${base_path}/.chrome_pid`
kill -9 ${chrome_pid}
rm -rf ${base_path}/.tmp/*
