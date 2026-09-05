var CONFIG_URL = "{{BASE_URL}}";

function getCookie(name) {
  var cookieStr;
  try {
    cookieStr = localCookie.getCookie() || "";
  } catch (e) {
    return "";
  }
  var parts = cookieStr.split(";");
  for (var i = 0; i < parts.length; i++) {
    var kv = parts[i].split("=");
    if (kv.length && kv[0].trim() === name) {
      return kv.slice(1).join("=").trim();
    }
  }
  return "";
}

function refreshAccess() {
  try {
    var csrf = getCookie("csrf_token");
    var headers = {};
    if (csrf) headers["X-CSRF-Token"] = csrf;
    var text = Http.post(CONFIG_URL + "/api/v1/auth/refresh")
      .headers(headers)
      .body("")
      .string("UTF-8");
    return !!(text && JSON.parse(text).status === true);
  } catch (e) {
    return false;
  }
}

function authHeader() {
  var token = getCookie("access_token");
  if (!token) {
    if (refreshAccess()) {
      token = getCookie("access_token");
    }
  }
  if (token) {
    return { Authorization: "Bearer " + token };
  }
  return {};
}

function execute(data) {
  // data is the stream URL from audio_chap
  return Response.success({
    type: "native",
    data: data,
    host: CONFIG_URL,
    headers: authHeader(),
  });
}
