<?php if (!empty($host->vm)) { ?>
  <?php foreach ($host->vm as $vm) { ?>
  <?php $vm_counter++; ?>

  <!-- INFO -->
  <div
    id="vm_container_<?php echo $vm_counter ?>" 
    class="searchable" 
    search-string="<?php echo $vm->hostname ?><?php echo $vm->ip ?>"
  >
    <div 
      id="vm_<?php echo $vm_counter ?>" 
      class="d-flex bd-highlight mb-3 align-items-center rounded shadow-sm ms-5 div-hover-grey searchable"
      search-string="<?php echo $vm->hostname ?>"
    >
      <div class="p-3 pe-0 bd-highlight">
        <svg class="bd-placeholder-img flex-shrink-0 me-2 rounded" width="45" height="45" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Placeholder: 32x32" preserveAspectRatio="xMidYMid slice" focusable="false">
          <title><?php echo $vm->hostname ?></title>
          <rect width="100%" height="100%" fill="#399ca1"/>
          <text x="50%" y="50%" fill="#ffffff" dy=".4em" class="svg-text">vm</text>
        </svg>
      </div>
      <div class="p-3 ps-0 bd-highlight">
        <div class="lh-1">
          <h1 class="h6 mb-2 lh-1 text-muted">
            <?php echo $vm->hostname ?>
          </h1>
          <small class="mb-0 lh-1 text-muted">
            <?php echo $vm->ip ?>:<?php echo $vm->port ?>
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
            '<?php echo $vm->user ?>',
            '<?php echo $vm->ip ?>',
            '<?php echo $vm->port ?>'
          )"
        >
          <span class="material-icons violet-icon" title="SSH Connect">terminal</span>
        </a>

        <!-- Remote Filesystem -->
        <a
          role="button" 
          onclick="callSshApi(
            'remotefs',
            '<?php echo $vm->user ?>',
            '<?php echo $vm->ip ?>',
            '<?php echo $vm->port ?>'
          )"
          >
          <span class="material-icons violet-icon" title="Open remote folder">folder_open</span>
        </a>

        <!-- SSH Copy ID -->
        <a
          role="button" 
          onclick="callSshApi(
            'ssh_copy_id',
            '<?php echo $vm->user ?>',
            '<?php echo $vm->ip ?>',
            '<?php echo $vm->port ?>'
          )"
        >
          <span class="material-icons violet-icon" title="SSH Copy Key">vpn_key</span>
        </a>

        <!-- Delete button -->
        <a
          href="#" data-bs-toggle="modal"
          data-bs-target="#modalDel"
          hostname="<?php echo $vm->hostname ?>"
          onclick="deleteHost(this)"
        >
          <span class="material-icons grey-icon" title="Delete">delete_forever</span>
        </a>

      </div>
      <!-- / COMMANDS -->

    </div>
  </div>

  <?php } ?>
<?php } ?>