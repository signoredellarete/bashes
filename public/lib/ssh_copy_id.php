<?php 
  function ssh_copy_id () {
    if (isset($_REQUEST['ssh_copy_id'])) {
      $host = $_REQUEST['host'];
      $port = $_REQUEST['port'];
      $command='gnome-terminal --tab --maximize --title="'.$host.'" --working-directory="/home/fabrizio" -- bash -c \'/bin/ssh-copy-id -p '.$port.' '.$host.'\'';
      $output=null;
      $retval=null;
      exec($command, $output, $retval);
      header("location: /");
    }
  }
  ssh_copy_id();
?>