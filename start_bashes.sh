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

# Start a php embedded web server instance
nohup /usr/bin/php -S 127.0.0.1:${phpws_port} -t ${base_path}/public > ${base_path}/log/bashes.log 2>&1 &
echo $! > ${base_path}/.pid

# Start new instance of google-chrome in incognito mode
nohup /usr/bin/google-chrome --incognito --no-first-run --no-gpu --user-data-dir=${base_path}/.tmp/ 127.0.0.1:${phpws_port} > ${base_path}/log/chrome.log 2>&1 &
echo $! > ${base_path}/.chrome_pid

# Read pids
pid=`cat ${base_path}/.pid`
chrome_pid=`cat ${base_path}/.chrome_pid`

# Monitor php web server and google-chrome started instances
# Once one of these two process become inactive the monitor wil kill the other one as well
while
  true
do

  chrome_procs=`ps -fe|grep ${chrome_pid}|grep -v grep|wc -l`
  bashes_procs=`ps -fe|grep ${pid}|grep -v grep|wc -l`

  if [ ${chrome_procs} -eq 0 ];then
    kill -9 ${pid}
    rm -rf ${base_path}/.tmp/*
    rm ${base_path}/.chrome_pid
    rm ${base_path}/.pid
  fi

  if [ ${bashes_procs} -eq 0 ];then
    kill -9 ${chrome_pid}
    rm -rf ${base_path}/.tmp/*
    rm ${base_path}/.chrome_pid
    rm ${base_path}/.pid
  fi

  sleep 3
done
