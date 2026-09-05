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

function isAuthError(res) {
  var m = res && res.message;
  return (
    m === "Authentication required" ||
    m === "Invalid credentials" ||
    m === "Missing or malformed JWT" ||
    m === "Invalid or expired JWT" ||
    m === "Invalid or missing token" ||
    m === "Token has been invalidated"
  );
}

function fetchJson(url) {
  var text = Http.get(url).string("UTF-8");
  if (!text) return null;
  var res = JSON.parse(text);
  if (res.status === false) {
    if (refreshAccess()) {
      text = Http.get(url).string("UTF-8");
      if (text) res = JSON.parse(text);
    } else if (isAuthError(res)) {
      res.needsLogin = true;
    }
  }
  return res;
}

function loginError() {
  return Response.error(
    "Cần đăng nhập NovelHub: mở trình duyệt trong app, truy cập " +
      CONFIG_URL +
      "/login, đăng nhập rồi quay lại thử",
  );
}

function execute() {
  return Response.success([
    {
      title: "Mới thêm gần đây",
      input: CONFIG_URL + "/api/v1/vbook/books?sort=created",
      script: "gen.js",
    },
    {
      title: "Sách xem nhiều",
      input: CONFIG_URL + "/api/v1/vbook/books?sort=hot",
      script: "gen.js",
    },
    {
      title: "Sách ngẫu nhiên",
      input: CONFIG_URL + "/api/v1/vbook/books?sort=random",
      script: "gen.js",
    },
    {
      title: "Sách mới cập nhật",
      input: CONFIG_URL + "/api/v1/vbook/books?sort=updated",
      script: "gen.js",
    },
    {
      title: "Tất cả sách",
      input: CONFIG_URL + "/api/v1/vbook/books",
      script: "gen.js",
    },
  ]);
}
