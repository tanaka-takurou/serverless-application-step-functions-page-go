var OpenModal = function() {
  $('.large.modal').modal('show');
}
var CloseModal = function() {
  $('.large.modal').modal('hide');
}
var toBase64 = function(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);
    reader.onload = () => resolve(reader.result);
    reader.onerror = error => reject(error);
  });
}
var OnConverted = function() {
  return function(v) {
    App.imgdata = v;
    $('#preview').attr('src', v);
  }
}
var UploadImage = function(elm) {
  if (!!App.imgdata) {
    $(elm).addClass("disabled");
    PutImage();
  }
}
var PutImage = function() {
  const file = $('#image').prop('files')[0];
  App.extension = GetExtension(file.name);
  const data = {action: 'upload', filename: file.name, filedata: App.imgdata};
  Request(data, (res)=>{
    App.sid = res.message;
    App.progress = 'convert_jpg';
    CheckProgress();
    $("#info").removeClass("hidden").addClass("visible");
    ScrollBottom();
    setTimeout(function() {
      CheckStatus();
    }, 3000);
  }, (e)=>{
    console.log(e.responseJSON.message);
  });
}
var ChangeImage = function() {
  const file = $('#image').prop('files')[0];
  toBase64(file).then(OnConverted());
}
var CheckStatus = function() {
  const data = {action: 'checkstatus', id: App.sid};
  Request(data, (res)=>{
    if (res.message == 'RUNNING') {
      setTimeout(function() {
        CheckStatus();
      }, 30000);
    } else if (res.message == 'SUCCEEDED') {
      console.log(res.message);
    } else {
      console.log(res.message);
      $("#warning").text(res.message).removeClass("hidden").addClass("visible");
      ScrollBottom();
    }
  }, (e)=>{
    console.log(e.responseJSON.message);
  });
}

var Request = function(data, callback, onerror) {
  $.ajax({
    type:          'POST',
    dataType:      'json',
    contentType:   'application/json',
    scriptCharset: 'utf-8',
    data:          JSON.stringify(data),
    url:           App.url
  })
  .done(function(res) {
    callback(res);
  })
  .fail(function(e) {
    onerror(e);
  });
};

var CheckProgress = function() {
  if (!App.sid) {
    $("#warning").text("ID is Empty").removeClass("hidden").addClass("visible");
    return false;
  }
  var url = App.bucket + App.sid.substr(0, 4) + '-' + App.sid.substr(4, 2) + '-' + App.sid.substr(6, 2) + '-' + App.sid.substr(8, 2) + '-' + App.sid.substr(10, 2) + '/' + App.sid;
  switch (App.progress){
  case "convert_jpg":
    url += "_convert.jpg"
    break;
  case "convert_png":
    url += "_convert.png"
    break;
  case "icon_200":
    url += "_icon_200.png"
    break;
  case "icon_300":
    url += "_icon_300.png"
    break;
  case "thumbnail_960_540":
    url += "_thumbnail_960_540." + App.extension
    break;
  case "thumbnail_1440_810":
    url += "_thumbnail_1440_810." + App.extension
    break;
  case "thumbnail_480_270":
    url += "_thumbnail_480_270." + App.extension
    break;
  }
  CheckExist(url, (res)=>{
      switch (App.progress){
      case "convert_jpg":
        App.progress = "convert_png";
        $("#img_convert_jpg_link").removeClass("active").removeClass("loader").attr('href', url);
        $("#img_convert_jpg").attr('src', url);
        ScrollBottom();
        CheckProgress();
        break;
      case "convert_png":
        App.progress = "icon_200";
        $("#img_convert_png_link").removeClass("active").removeClass("loader").attr('href', url);
        $("#img_convert_png").attr('src', url);
        ScrollBottom();
        CheckProgress();
        break;
      case "icon_200":
        App.progress = "icon_300";
        $("#img_icon_200_link").removeClass("active").removeClass("loader").attr('href', url);
        $("#img_icon_200").attr('src', url);
        ScrollBottom();
        CheckProgress();
        break;
      case "icon_300":
        App.progress = "thumbnail_960_540";
        $("#img_icon_300_link").removeClass("active").removeClass("loader").attr('href', url);
        $("#img_icon_300").attr('src', url);
        ScrollBottom();
        CheckProgress();
        break;
      case "thumbnail_960_540":
        App.progress = "thumbnail_1440_810";
        $("#img_thumbnail_960_540_link").removeClass("active").removeClass("loader").attr('href', url);
        $("#img_thumbnail_960_540").attr('src', url);
        ScrollBottom();
        CheckProgress();
        break;
      case "thumbnail_1440_810":
        App.progress = "thumbnail_480_270";
        $("#img_thumbnail_1440_810_link").removeClass("active").removeClass("loader").attr('href', url);
        $("#img_thumbnail_1440_810").attr('src', url);
        ScrollBottom();
        CheckProgress();
        break;
      case "thumbnail_480_270":
        App.progress = "finish";
        $("#img_thumbnail_480_270_link").removeClass("active").removeClass("loader").attr('href', url);
        $("#img_thumbnail_480_270").attr('src', url);
        ScrollBottom();
        break;
      }
  }, (e)=>{
    setTimeout(function() {
      CheckProgress();
    }, 2000);
  });
};

var CheckExist = function(url, callback, onerror) {
  $.ajax({
    type: 'HEAD',
    url:  url
  })
  .done(function(res) {
    callback(res);
  })
  .fail(function(e) {
    onerror(e);
  });
};

var ScrollBottom = function() {
  var bottom = document.documentElement.scrollHeight - document.documentElement.clientHeight;
  window.scroll(0, bottom);
}

var GetExtension = function(str) {
  var re = /(?:\.([^.]+))?$/;
  extension = re.exec(str)[1];
  if (extension == "jpeg") {
    extension = "jpg";
  }
  return extension;
}

var App = { sid: '', progress: '', extension: '', imgdata: null, url: location.origin + {{ .ApiPath }}, bucket: {{ .Bucket }} };
