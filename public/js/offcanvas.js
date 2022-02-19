function getOffset(el) {
  const rect = el.getBoundingClientRect();
  return {
    left: rect.left + window.scrollX,
    top: rect.top + window.scrollY
  };
}

function placeDiv(el, x_pos, y_pos) {
  el.style.position = "absolute";
  el.style.left = x_pos+'px';
  el.style.top = y_pos+'px';
}

function deleteHost(el){
  var hostname = el.getAttribute("hostname");
  document.getElementById("delete_host").value = hostname;
  document.getElementById("label_delete_host").innerHTML = hostname;
  console.log(hostname);
}

function displayIsNone(el) {
  if (window.getComputedStyle(el).display === "none") {
    return true;
  }
}


function showAddSubsystem(el){
  var host_div_id = el.id;
  //console.log(host_div_id);
  var host_div_offset = getOffset(el);
  var host_div_left = host_div_offset['left'];
  var host_div_top = host_div_offset['top'];
  //console.log(host_div_left);
  //console.log(host_div_top);
  var hostname = el.getAttribute("hostname");
  var add_div = document.getElementById("add_subsystem_for_" + host_div_id);
  placeDiv(add_div, host_div_left + 27, host_div_top + 50);
}

function addSubsystem(el) {
  var server = el.getAttribute("ref_server");
  document.getElementById("modal-add-new-subsystem-var").innerHTML = server;
  document.getElementById("new-subsystem-server-name").value = server;
}

/* Search */
function search() {
  var searchbox = document.getElementById("searchbox");
  searchbox.addEventListener("keyup", (k) => {
    console.log("input");
  });
  var search = searchbox.value.toLowerCase();
  search_len = search.length;

  var items_to_seach = document.getElementsByClassName("searchable");
  if (search_len == 0){
    for (const el in items_to_seach) {

    }
  }
}
search();

/*
function hideAddSubsystem(el){
  var host_div_id = el.getAttribute("id");
  var hostname = el.getAttribute("hostname");
  var add_div = document.getElementById("add_subsystem_for_" + host_div_id);
  add_div.style.display = "none";
}
*/

