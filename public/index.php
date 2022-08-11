<?php 
  require_once("lib/func.php");
  $json_hosts = file_get_contents("db/hosts.json");
  $hosts = json_decode($json_hosts);
  ssh_connect();
  ssh_copy_id();
?>
<?php require_once("header.php"); ?>

<main class="container">

  <div class="d-flex bd-highlight mb-3 align-items-center text-white bg-purple rounded shadow-sm">
    <div class="p-3 pe-0 bd-highlight">
      <a href="/">
        <img class="me-3" src="img/bashes.png" alt="" width="48">
      </a>
    </div>
    <div class="p-3 ps-0 bd-highlight">
      <div class="lh-1">
        <h1 class="h6 mb-0 text-white lh-1">Bashes</h1>
        <small>CLI rules!</small>
      </div>
    </div>
    <div class="ms-auto p-2 pe-3">
      <div class="input-group input-group">
        <input onkeyup="search(this.value)" id="searchbox" type="text" class="form-control searchbox" placeholder="Search...">
        <button onclick="resetInput('searchbox')" id="searchResetButton" class="btn btn-light" type="button">Reset</button>
      </div>
    </div>
  </div>

  <div class="row">
    <div class="col d-flex align-items-end flex-column">
      <button 
        title="Add host" 
        class="btn violet-icon d-flex" 
        data-bs-toggle="modal" 
        data-bs-target="#modalAddHost"
      >
        <i class="material-icons">add_circle_outline</i><span class="ms-1">Add host</span>
      </button>
    </div>
  </div>

  <div class="my-3 p-3 bg-body rounded shadow-sm">
    <h6 class="border-bottom pb-2 mb-0 text-center">Servers, Linux Containers, Virtual Machines, Docker containers </h6>

    <?php $hosts_counter = 0; ?>
    <?php $lxc_counter = 0; ?>
    <?php $vm_counter = 0; ?>
    <?php $docker_counter = 0; ?>

    <?php foreach ($hosts as $host) { ?>
      <?php $hosts_counter++; ?>
      <?php require("hosts.php") ?>
    <?php } ?>


    <small class="d-block text-end mt-3">
      <a href="#">All suggestions</a>
    </small>
  </div>
</main>

<?php require_once("modal_add_host.php") ?>
<?php require_once("modal_add_subsystem.php") ?>
<?php require_once("modal_del.php") ?>
<?php require_once("footer.php") ?>