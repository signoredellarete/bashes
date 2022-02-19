<form class="form-floating" action="/lib/add_host.php" method="POST">
  <input type="hidden" name="add_host">

  <div class="form-floating mb-3">
    <input type="text" class="form-control" id="new-hostname" name="hostname" placeholder="Hostname" value="" required>
    <label for="new-hostname">Hostname</label>
  </div>

  <div class="form-floating mb-3">
    <input type="text" class="form-control" id="new-ip" name="ip" placeholder="IP" required>
    <label for="new-ip">IP address</label>
  </div>

  <div class="form-floating mb-3">
    <input type="text" class="form-control" id="new-port" name="port" placeholder="Port" required>
    <label for="new-port">Port</label>
  </div>

  <div class="form-floating mb-3">
    <input type="text" class="form-control" id="new-user" name="user" placeholder="User" required>
    <label for="new-user">User</label>
  </div>

  <div class="input-group input-group-lg d-grid gap-2">
    <button role="submit" class="btn violet-bg text-white">SAVE</button>
  </div>

</form>