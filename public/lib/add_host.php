<?php 
function add_host() {
  if (isset($_REQUEST['add_host'])) {
    unset($_REQUEST['add_host']);
    $json_hosts = file_get_contents("../db/hosts.json");

    $hosts = json_decode($json_hosts, true);

    if (is_null($hosts)) {
      $hosts = array();
    }

    $hostname = $_REQUEST['hostname'];
    $ip = $_REQUEST['ip'];
    $port = $_REQUEST['port'];
    $user = $_REQUEST['user'];

    unset($_REQUEST['hostname']);
    unset($_REQUEST['ip']);
    unset($_REQUEST['port']);
    unset($_REQUEST['user']);

    $new_json_host = '{
      "hostname":"'.$hostname.'",
      "ip":"'.$ip.'",
      "port":"'.$port.'",
      "user":"'.$user.'",
      "lxc": [],
      "vm": [],
      "docker": []
    }';

    $new_host = json_decode($new_json_host, true);

    array_push($hosts, $new_host);

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