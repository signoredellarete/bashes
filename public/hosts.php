<!-- INFO -->

<div 
  id="host_<?php echo $hosts_counter ?>" 
  hostname="<?php echo $host->hostname ?>" 
  class="d-flex bd-highlight mb-3 align-items-center rounded shadow-sm div-hover-grey"
  onmouseover="showAddSubsystem(this)" 
>
  <div class="p-3 pe-0 bd-highlight">
    <svg class="bd-placeholder-img flex-shrink-0 me-2 rounded" width="45" height="45" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Placeholder: 32x32" preserveAspectRatio="xMidYMid slice" focusable="false">
      <title><?php echo $host->hostname ?></title>
      <rect width="100%" height="100%" fill="#6f42c1"/>
      <text x="50%" y="50%" fill="#ffffff" dy=".4em" class="svg-text">host</text>
    </svg>
  </div>
  <!-- Add Subsystem Button -->
  <div 
    id="add_subsystem_for_host_<?php echo $hosts_counter ?>" 
    host_div_ref="host_<?php echo $hosts_counter ?>" 
    ref_server="<?php echo $host->hostname ?>" 
    class="add-subsystem" 
    title="Add subsystem" 
    data-bs-toggle="modal" 
    data-bs-target="#modalAddSubsystem" 
    onclick="addSubsystem(this)"
  >
    <span class="material-icons fuksia-icon p-0 m-0">add_box</span>
  </div>
  <div class="p-3 ps-0 bd-highlight">
    <div class="lh-1">
      <h1 class="h6 mb-2 lh-1 text-muted">
        <?php echo $host->hostname ?>
      </h1>
      <small class="mb-0 lh-1 text-muted">
        <?php echo $host->ip ?>:<?php echo $host->port ?>
      </small>
    </div>
  </div>

  <!-- COMMANDS -->
  <div class="ms-auto p-2 pe-3">

    <!-- Connect SSH -->
    <a
      role="button" 
      onclick="callSshApi(
        'connect',
        '<?php echo $host->user ?>',
        '<?php echo $host->ip ?>',
        '<?php echo $host->port ?>'
      )"
    >
      <span class="material-icons violet-icon" title="SSH Connect">terminal</span>
    </a>
    
    <!-- SSH Copy ID -->
    <a
      role="button" 
      onclick="callSshApi(
        'ssh_copy_id',
        '<?php echo $host->user ?>',
        '<?php echo $host->ip ?>',
        '<?php echo $host->port ?>'
      )"
    >
      <span class="material-icons violet-icon" title="SSH Copy Key">vpn_key</span>
    </a>
    
    <!-- Proxmox link -->
    <a
      href="https://<?php echo $host->ip ?>:8006"
      target="_blank"
    >
        <span class="material-icons violet-icon" title="Proxmox">filter_drama</span>
    </a>

    <!-- Delete button -->
    <a
      href="#" data-bs-toggle="modal"
      data-bs-target="#modalDel"
      hostname="<?php echo $host->hostname ?>"
      onclick="deleteHost(this)"
    >
      <span class="material-icons grey-icon" title="Delete">delete_forever</span>
    </a>

  </div>
  <!-- / COMMANDS -->

</div>

<?php require("lxc.php"); ?>
<?php require("vm.php"); ?>
<?php require("docker.php"); ?>