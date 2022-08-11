<?php if (!empty($host->docker)) { ?>
  <?php foreach ($host->docker as $docker) { ?>
  <?php $docker_counter++; ?>

  <!-- INFO -->
  <div
    id="docker_container_<?php echo $docker_counter ?>" 
    class="searchable" 
    search-string="<?php echo $docker->hostname ?><?php echo $docker->ip ?>"
  >
    <div 
      id="docker_<?php echo $docker_counter ?>" 
      class="d-flex bd-highlight mb-3 align-items-center rounded shadow-sm ms-5 div-hover-grey searchable" 
      search-string="<?php echo $docker->hostname ?>"
    >
      <div class="p-3 pe-0 bd-highlight">
        <svg class="bd-placeholder-img flex-shrink-0 me-2 rounded" width="45" height="45" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Placeholder: 32x32" preserveAspectRatio="xMidYMid slice" focusable="false">
          <title><?php echo $docker->hostname ?></title>
          <rect width="100%" height="100%" fill="#651c32"/>
          <text x="50%" y="50%" fill="#ffffff" dy=".4em" class="svg-text">docker</text>
        </svg>
      </div>
      <div class="p-3 ps-0 bd-highlight">
        <div class="lh-1">
          <h1 class="h6 mb-2 lh-1 text-muted">
            <?php echo $docker->hostname ?>
          </h1>
          <small class="mb-0 lh-1 text-muted">
            <?php echo $docker->ip ?>:<?php echo $docker->port ?>
          </small>
        </div>
      </div>
      <!-- COMMANDS -->
      <div class="ms-auto p-2 pe-3">
        <a href="/?connect=1&host=<?php echo $docker->user."@".$docker->ip ?>&port=<?php echo $docker->port ?>"><span class="material-icons violet-icon" title="Connect">terminal</span></a>
        <a href="/?ssh_copy_id=1&host=<?php echo $docker->user."@".$docker->ip ?>&port=<?php echo $docker->port ?>"><span class="material-icons violet-icon" title="Copy SSH Key">vpn_key</span></a>
        <a href="#" data-bs-toggle="modal" data-bs-target="#modalDel" hostname="<?php echo $docker->hostname ?>" onclick="deleteHost(this)"><span class="material-icons grey-icon" title="Delete">delete_forever</span></a>

      </div>
    </div>
  </div>

  <?php } ?>
<?php } ?>