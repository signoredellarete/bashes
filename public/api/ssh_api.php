<?php
  header('Content-Type: application/json');
  /*
  +-------------------------
  | 
  | ssh_api Required vars: func, user, ip, port
  |
  +-------------------------
  */

  $input_data = json_decode(file_get_contents('php://input'), true);
  $required_vars = ['func', 'user', 'ip', 'port'];

  foreach ($required_vars as $var) {
    if (!isset($input_data[$var]) || $input_data[$var] == '' || $input_data[$var] == null) {
      echo json_encode('Missing parameters');
      exit;
    }
  }

  $command = '';
  $user = $input_data['user'];
  $ip = $input_data['ip'];
  $port = $input_data['port'];

  if ($input_data['func'] == 'connect') {
    $func = '/bin/ssh';
    $command='gnome-terminal --title="'.$user.'@'.$ip.'" --working-directory="/home/sid" -- bash -c \''.$func.' -p '.$port.' '.$user.'@'.$ip.'\'';
  }

  if ($input_data['func'] == 'remotefs') {
    $func = '/bin/ssh';
    $command = '/usr/bin/nemo ssh://'.$user.'@'.$ip.':'.$port;
  }

  if ($input_data['func'] == 'ssh_copy_id') {
    $func = '/bin/ssh-copy-id';
    $command='gnome-terminal --title="'.$user.'@'.$ip.'" --working-directory="/home/sid" -- bash -c \''.$func.' -p '.$port.' '.$user.'@'.$ip.'\'';
  }



  $output=null;
  $retval=null;
  exec($command, $output, $retval);

  echo json_encode($retval);
  exit;
?>