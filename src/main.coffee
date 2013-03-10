ulurl = "/upload"
statusurl = "/status/"
uploading = false
ulid = -1
lastId = 0

$(document).ready ->
  $("#file").filestyle()
  lid = $("#fileList").children(":first")?.attr('id').replace(/file/, '')
  lastId = parseInt(lid, 10) or -1
  $.jsonRPC.setup
    endPoint: '/rpc'
    namespace: 'UploadService'
  doUpdate()

stopLoad = ->
  iframe = document.getElementById('ulframe')
  if iframe.contentWindow.document.execCommand
    iframe.contentWindow.document.execCommand('Stop')
  else
    iframe.contentWindow.stop()
  $(iframe).attr 'src', 'javascript:false'

finishUpload = ->
  uploading = false
  $("#progressModal").modal('hide')
  $("#progressBar").css 'width', '100%'

doUpload = ->
  return false if not $("#file").val()?.length
  $.jsonRPC.request 'NewUpload',
    params: []
    success: (r) ->
      ulid = r.result.Ulid
      $("#upload").attr 'action', ulurl + "?ul=" + ulid
      $("#upload").submit()
      uploading = true
      $("#progressModal").modal('show')
      window.setTimeout(doProgress, 500)
    error: (r) ->
      error = r.error
      alert(error)
  return false

doProgress = ->
  return false if not uploading
  requestObj = {Ulid: ulid}
  $.jsonRPC.request 'Status',
    params: [requestObj]
    success: (r) ->
      if not r.result.Status.Started
        return false
      if r.result.Status.Completed
        finishUpload()
        return true
      u = r.result.Status.Uploaded
      t = r.result.Status.Total
      p = "" + Math.floor(u * 100 / t) + "%"
      $("#progressBar").css 'width', p
      window.setTimeout(doProgress, 500) if uploading
    error: (r) ->
      error = r.error
      console.log(error)
      window.setTimeout(doProgress, 500) if uploading
  return true

doCancel = ->
  return false if not uploading
  stopLoad()
  finishUpload()
  return true

doUpdate = ->
  requestObj = {LastId: lastId}
  $.jsonRPC.request 'Updates',
    params: [requestObj]
    success: (r) ->
      if lastId < r.result.LastId
        lastId = r.result.LastId
        $("#fileList").prepend $.trim(r.result.Updates)
  window.setTimeout(doUpdate, 5000)
  return true
