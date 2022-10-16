#!/bin/bash
SCRIPT=$(readlink -f "$0")
base_path=$(dirname "$SCRIPT")
start_script=${base_path}/start_bashes.sh

echo base_path=$(dirname "$SCRIPT") > ${base_path}/.env
echo start_script=${base_path}/start_bashes.sh >> ${base_path}/.env
echo desktop_icon=${base_path}/icons/bashes4.png >> ${base_path}/.env

source ${base_path}/.env

# Listening port
while true
  do
  default_phpws_port=64888
  echo -n "Choose appplicazion listening port (from 1024 to 65534), default is [64888] "
  read -r phpws_port

  if [ -z ${phpws_port} ];then
    phpws_port=${default_phpws_port}
    echo phpws_port=${phpws_port} >> ${base_path}/.env
    break
  else
    if [[ ${phpws_port} -gt 1024 ]] && [[ ${phpws_port} -lt 65534 ]];then
      echo phpws_port=${phpws_port} >> ${base_path}/.env
      break
    else
      echo "Invalid port!"
    fi
  fi
done


# Desktop launcher
if [ ! -f ~/Desktop/bashes.desktop ];then
  while true
    do
    echo -n "Do you want to create a Desktop launcher? [Y/n] "
    read -r "DESK_LAUNCH_INSTALL"

    case ${DESK_LAUNCH_INSTALL} in
      "Y"|"y"|"")
        DESK_LAUNCH_INSTALL="Y"
        break
        ;;

      "N"|"n")
        DESK_LAUNCH_INSTALL="N"
        break
        ;;

      *)
        echo "Invalid selection!"
        ;;
    esac
  done

  if [ ${DESK_LAUNCH_INSTALL} = "Y" ];then
    sed -e s/"Exec=.*"/"Exec=$(echo ${start_script}|sed -e s/'\/'/'\\\/'/g)"/g ${base_path}/"bashes.desktop" |sed -e s/"Icon=.*"/"Icon=$(echo ${desktop_icon}|sed -e s/'\/'/'\\\/'/g)"/g > ~/Desktop/bashes.desktop
    chmod +x ~/Desktop/bashes.desktop
  fi
fi

# Browser
browsers="google-chrome-stable google-chrome firefox chromium epiphany epiphany-browser"
for i in $browsers
  do
  which ${i} 2>&1 > /dev/null
  if [ $? -eq 0 ];then
    echo browser=`which ${i}` >> ${base_path}/.env
    break
  fi
done

# File explorer
explorers="nemo nautilus"
for i in $explorers
  do
  which ${i} 2>&1 > /dev/null
  if [ $? -eq 0 ];then
    echo explorer=`which ${i}` >> ${base_path}/.env
    break
  fi
done

# Terminal emulator
terminals="gnome-terminal xterm"
for i in $terminals
  do
  which ${i} 2>&1 > /dev/null
  if [ $? -eq 0 ];then
    echo terminal=`which ${i}` >> ${base_path}/.env
    break
  fi
done


# .tmp directory
if [ -d ${base_path}/.tmp ];then
  rm -rf ${base_path}/.tmp
  mkdir ${base_path}/.tmp
else
  mkdir ${base_path}/.tmp
fi

# log directory
if [ ! -d ${base_path}/log ];then
  mkdir ${base_path}/log
fi

# db directory
if [ ! -d ${base_path}/public/db ]; then
  mkdir ${base_path}/public/db
fi

# hosts.json file
if [ ! -f ${base_path}/public/db/hosts.json ];then
  touch ${base_path}/public/db/hosts.json
fi

source ${base_path}/.env
