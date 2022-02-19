<?php 
function delete_host() {
  if (isset($_REQUEST['delete_host'])) {
    $hostname = $_REQUEST['delete_host'];
    unset($_REQUEST['delete_host']);

    $json_hosts = file_get_contents("../db/hosts.json");
    $hosts = json_decode($json_hosts, true);
    $i = 0;
    foreach ($hosts as $i => $host) {
      if($host['hostname'] == $hostname) {
        echo "<pre>";
        print_r($host);
        echo "</pre>";
        unset($hosts[$i]);
      }
      $subsystems = ["lxc","vm","docker"];
      foreach ($subsystems as $subsystem) {
        if (!empty($host[$subsystem])) {
          foreach ($host[$subsystem] as $$subsystem => $sub) {
            if ($sub['hostname'] == $hostname) {
              unset($hosts[$i][$subsystem][$$subsystem]);
            }
          }
        }
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
delete_host();
?>