<div 
  id="modalAddSubsystem" 
  class="modal fade" 
  tabindex="-1" 
  aria-labelledby="modalAddSubsystemLabel" 
  aria-hidden="true"
>
  <div class="modal-dialog modal-dialog-centered">
    <div class="modal-content">
      <div class="modal-header">
        <h5 class="modal-title" id="modalAddSubsystemLabel">
          Add new subsystem for server: 
          <span 
            id="modal-add-new-subsystem-var" 
            class="fw-bold"
          >
            <!-- JS -->
          </span>
        </h5>
        <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
      </div>
      <div class="modal-body">
        <?php require("form_new_subsystem.php"); ?>
      </div>
    </div>
  </div>
</div>