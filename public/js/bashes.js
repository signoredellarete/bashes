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
  // console.log(hostname); /debug
}

function displayIsNone(el) {
  if (window.getComputedStyle(el).display === "none") {
    return true;
  }
}

function showAddSubsystem(el){
  var host_div_id = el.id;
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
var itemsToSearchIn = document.getElementsByClassName("searchable");
var searchStrings = [];
for (const el of itemsToSearchIn) {
  searchStrings.push({
    id: el.id,
    text: el.getAttribute("search-string").toLowerCase()
  });
}

// console.log(searchStrings); //debug

function search(val) {

  if (val.length == 0){
    for (const el of searchStrings) {
      document.getElementById(el.id).style.display = "block";
    }
  } else {

    // console.log(val); //debug

    for (const el of searchStrings) {
      if (!el.text.includes(val.toLowerCase())) {
        // console.log(); //debug
        document.getElementById(el.id).style.display = "none";
      } else {
        document.getElementById(el.id).style.display = "block";
      }
    }
  }

}

function resetInput(el) {
  document.getElementById(el).value = '';
  document.getElementById(el).dispatchEvent(new Event('keyup'));
}


function callSshApi(func, user, ip, port) {
  fetch('/api/ssh_api.php', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      'func': func,
      'user': user,
      'ip': ip,
      'port': port
    })
  }).then(response => response.json())
    // .then(data => {
    //     // document.querySelector('#result' ).innerText = data.encoded;
    //     console.log(data);
    // });
}