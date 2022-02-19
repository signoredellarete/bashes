<div class="modal fade" id="modalDel" tabindex="-1" aria-labelledby="modalDelLabel" aria-hidden="true">
  <div class="modal-dialog">
    <div class="modal-content">
      <div class="modal-header">
        <h5 class="modal-title" id="modalDelLabel">Delete</h5>
        <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
      </div>
      <div class="modal-body">
        Are you sure you want to delete <strong><span id="label_delete_host"><!-- JS --></span></strong> ?
        <form id="form_del" action="/lib/delete_host.php" method="post">
          <input id="delete_host" type="hidden" name="delete_host" value="">
        </form>
      </div>
      <div class="modal-footer">
        <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
        <button type="button" class="btn violet-bg text-white" onclick="submitDelForm()">Confirm</button>
      </div>
    </div>
  </div>
</div>
<script>
  function submitDelForm() {
    delForm = document.getElementById("form_del").submit();
  }
</script>