# bashes
Bashes is a local web application for linux aimed at helping system administrators to group servers, virtual machines and containers in an ordered list that provides some buttons to open ssh connections, copy ss keys and open other management web apps like Proxmox. A kind of advanced hosts file.

**NOT READY FOR PRODUCTION**

...

## Get started

### Prerequisites
- Linux Operating System (Tested on Linux Mint 21 Vanessa)
- Desktop environment
- git
- php-cli (>= 7.4)
- google-chrome
- gnome-terminal
- nemo (File manager)

### Installation
- Clone this git repository
```
git clone https://github.com/signoredellarete/bashes.git
```
- Enter the just cloned directory
```
cd bashes
```
- Launch installer
```
bash install.sh
```
- Launch application
```
bash start_bashes.sh &
```
If you have chosen to create a the desktop lancher during installation it will be possible to launch the application by double-clicking on the icon.

### Use
Bashes is very simple to use. You can add hosts and for each host you wil be able to create subsystems entries (VM, LXC, Docker).

Bashes use a json file for store all the data, this makes it simple to backup data, read data and to transfer the db from one Bashes installation to another.

**Functions available for hosts:**

- Add a subsystem
- Open an ssh terminal pointing to the specified ip address and port
- Open the remote user's home directory using the nemo graphical file manager
- Copy your ssh key to the remote server (typing the password)
- Open the Proxmox Web console link on port 8006
- Remove the host

**Functions available for subsystems:**

- Open an ssh terminal pointing to the specified ip address and port
- Open the remote user's home directory using the nemo graphical file manager
- Copy your ssh key to the remote server (by typing in the password)
- Remove the host

