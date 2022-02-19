<?php 
function add_host() {
  print_r($_REQUEST);
  if (isset($_REQUEST['add_subsystem'])) {
    unset($_REQUEST['add_subsystem']);
    $json_hosts = file_get_contents("../db/hosts.json");
    $hosts = json_decode($json_hosts, true);

    $server = $_REQUEST['server'];
    $subsystem_type = $_REQUEST['type'];
    $hostname = $_REQUEST['hostname'];
    $ip = $_REQUEST['ip'];
    $port = $_REQUEST['port'];
    $user = $_REQUEST['user'];

    unset($_REQUEST['server']);
    unset($_REQUEST['type']);
    unset($_REQUEST['hostname']);
    unset($_REQUEST['ip']);
    unset($_REQUEST['port']);
    unset($_REQUEST['user']);

    $new_json_subsystem = '{
      "hostname":"'.$hostname.'",
      "ip":"'.$ip.'",
      "port":"'.$port.'",
      "user":"'.$user.'"
    }';

    $new_subsystem = json_decode($new_json_subsystem, true);

    foreach ($hosts as $key => $value) {
      if ($value['hostname'] == $server) {
        array_push($hosts[$key][$subsystem_type], $new_subsystem);
      }
    }

    $json = json_encode($hosts, JSON_PRETTY_PRINT);

    if (file_put_contents("../db/hosts.json", $json)){
      header("location: /");
    } else {
      echo "Oops! Error creating json file...";
    }

  }
}
add_host();
?>