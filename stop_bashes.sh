#!/bin/bash
SCRIPT=$(readlink -f "$0")
base_path=$(dirname "$SCRIPT")
pid=`cat ${base_path}/.pid`
kill -9 ${pid}
chrome_pid=`cat ${base_path}/.chrome_pid`
kill -9 ${chrome_pid}
