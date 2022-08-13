<?php 
  $json_hosts = file_get_contents("db/hosts.json");
  $hosts = json_decode($json_hosts);
?>
<?php require_once("header.php"); ?>

<main class="container" >

  <div class="nav bg-purple rounded shadow-sm p-3">

    <div class="col-5 mt-auto mb-auto">
      <div class="row" style="width: 200px;">
        <div class="col-4">
          <a href="/">
            <img src="img/bashes.png" alt="" width="48">
          </a>
        </div>
        <div class="col-8 mt-auto mb-auto">
          <h1 class="h6 mb-0 text-white lh-1">Bashes</h1>
          <small class="text-white">CLI rules!</small>
        </div>
      </div>
    </div> <!-- / col -->

    <div class="col-7 mt-auto mb-auto">
      <div class="input-group input-group">
        <input onkeyup="search(this.value)" id="searchbox" type="text" class="form-control searchbox" placeholder="Search...">
        <button onclick="resetInput('searchbox')" id="searchResetButton" class="btn btn-light" type="button">Reset</button>
      </div>
    </div> <!-- / col -->

  </div> <!-- / nav -->

  <div class="row">
    <div class="col-12 float-end">
      <a 
        title="Add host" 
        class="btn violet-icon d-flex float-end" 
        data-bs-toggle="modal" 
        data-bs-target="#modalAddHost"
      >
        <i class="material-icons">
          add_circle_outline
        </i>
        <span class="ms-1">
          Add host
        </span>
      </a>
    </div>
  </div>

  <div class="my-3 p-3 bg-body rounded shadow-sm">
    <div class="row">
      <div class="col-12">
        <h6 class="border-bottom pb-2 mb-0 text-center">Servers, Linux Containers, Virtual Machines, Docker containers </h6>
      </div>
    </div>


    <?php $hosts_counter = 0; ?>
    <?php $lxc_counter = 0; ?>
    <?php $vm_counter = 0; ?>
    <?php $docker_counter = 0; ?>

    <div class="row">
      <div class="col-12" style="overflow-y: auto; height: calc(100vh - 260px);">
        <?php foreach ($hosts as $host) { ?>
          <?php $hosts_counter++; ?>
          <?php require("hosts.php") ?>
        <?php } ?>
      </div>
    </div>

  </div>

</main>

<?php require_once("modal_add_host.php") ?>
<?php require_once("modal_add_subsystem.php") ?>
<?php require_once("modal_del.php") ?>
<?php require_once("footer.php") ?>