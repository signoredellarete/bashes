#!/bin/bash
base_path="/home/fabrizio/Documents/fabrizio/git/bashes"
pid=`cat ${base_path}/.pid`
kill -9 ${pid}
chrome_pid=`cat ${base_path}/.chrome_pid`
kill -9 ${chrome_pid}
