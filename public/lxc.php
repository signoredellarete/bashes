<?php if (!empty($host->lxc)) { ?>
  <?php foreach ($host->lxc as $lxc) { ?>
  <?php $lxc_counter++; ?>

  <!-- INFO -->
  <div
    id="lxc_container_<?php echo $lxc_counter ?>" 
    class="searchable" 
    search-string="<?php echo $lxc->hostname ?><?php echo $lxc->ip ?>"
  >
    <div 
      id="lxc_<?php echo $lxc_counter ?>" 
      class="d-flex bd-highlight mb-3 align-items-center rounded shadow-sm ms-5 div-hover-grey" 
    >
      <div class="p-3 pe-0 bd-highlight">
        <svg class="bd-placeholder-img flex-shrink-0 me-2 rounded" width="45" height="45" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Placeholder: 32x32" preserveAspectRatio="xMidYMid slice" focusable="false">
          <title><?php echo $lxc->hostname ?></title>
          <rect width="100%" height="100%" fill="#c14394"/>
          <text x="50%" y="50%" fill="#ffffff" dy=".4em" class="svg-text">LXC</text>
        </svg>
      </div>
      <div class="p-3 ps-0 bd-highlight">
        <div class="lh-1">
          <h1 class="h6 mb-2 lh-1 text-muted">
            <?php echo $lxc->hostname ?>
          </h1>
          <small class="mb-0 lh-1 text-muted">
            <?php echo $lxc->ip ?>:<?php echo $lxc->port ?>
          </small>
        </div>
      </div>
      <!-- COMMANDS -->
      <div class="ms-auto p-2 pe-3">
        <a href="/?connect=1&host=<?php echo $lxc->user."@".$lxc->ip ?>&port=<?php echo $lxc->port ?>"><span class="material-icons violet-icon" title="Connect">terminal</span></a>
        <a href="/?ssh_copy_id=1&host=<?php echo $lxc->user."@".$lxc->ip ?>&port=<?php echo $lxc->port ?>"><span class="material-icons violet-icon" title="Copy SSH Key">vpn_key</span></a>
        <a href="#" data-bs-toggle="modal" data-bs-target="#modalDel" hostname="<?php echo $lxc->hostname ?>" onclick="deleteHost(this)"><span class="material-icons grey-icon" title="Delete">delete_forever</span></a>
      </div>
    </div>
  </div>

  <?php } ?>
<?php } ?>