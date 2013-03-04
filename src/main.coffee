ulurl = "/upload"
statusurl = "/status/"
uploading = false
ulid = -1
lastUpdate = 0

$(document).ready ->
  $("#file").filestyle()
  lastUpdate = Math.floor(new Date().getTime() / 1000)
  doUpdate()

doUpload = ->
  return false if not $("#file").val()?.length
  $.ajax
    url: '/new_file'
    type: 'GET'
  .done (data) ->
    struct = JSON.parse(data)
    ulid = struct['Ulid']
    $("#upload").attr 'action', ulurl + "?ul=" + ulid
    $("#upload").submit()
  uploading = true
  return false

doProgress = ->
  return false if not uploading
  $.ajax  
    url: '/progress/' + ulid
    type: 'GET'
  .done (data) ->
    struct = JSON.parse(data)
    p = struct['Uled']
    t = struct['Total']
    complete = struct['Complete']
    prog = (+p) * 100.0 / (+t)
    $("pbar").style 'width', prog+'%'
    uploading = false if complete
  .always ->
    window.setTimeout doProgress, 1000 if uploading
  return true

doUpdate = ->
  $.ajax
    url: '/updates/' + lastUpdate
    type: 'GET'
  .done (data) ->
    $("#fileList").prepend($.trim(data))
  .always ->
    window.setTimeout doUpdate, 1000
  lastUpdate = Math.floor(new Date().getTime() / 1000)
  return true
