#!/bin/bash
base_path="/home/fabrizio/Documents/fabrizio/git/bashes"
nohup /usr/bin/php -S 127.0.0.1:8888 -t ${base_path}/public > ${base_path}/log/bashes.log 2>&1 &
echo $! > ${base_path}/.pid
if [ -d ${base_path}/.tmp ];then
rm -rf ${base_path}/.tmp
mkdir ${base_path}/.tmp
fi
if [ ! -d ${base_path}/public/db ];then
mkdir ${base_path}/.tmp
fi
nohup /usr/bin/google-chrome --incognito --no-first-run --no-gpu --user-data-dir=${base_path}/.tmp/ 127.0.0.1:8888 > ${base_path}/log/chrome.log 2>&1 &
echo $! > ${base_path}/.chrome_pid

pid=`cat ${base_path}/.pid`
chrome_pid=`cat ${base_path}/.chrome_pid`

while
true
do

chrome_procs=`ps -fe|grep ${chrome_pid}|grep -v grep|wc -l`
bashes_procs=`ps -fe|grep ${pid}|grep -v grep|wc -l`

if [ ${chrome_procs} -eq 0 ];then
kill -9 ${pid}
fi

if [ ${bashes_procs} -eq 0 ];then
kill -9 ${chrome_pid}
fi

sleep 3
done
