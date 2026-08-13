var CONFIG_URL = "{{BASE_URL}}";

function getCookie(name) {
    var cookieStr;
    try { cookieStr = localCookie.getCookie() || ""; } catch (e) { return ""; }
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
        var text = Http.post(CONFIG_URL + "/api/v1/auth/refresh").headers(headers).body("").string("UTF-8");
        return !!(text && JSON.parse(text).status === true);
    } catch (e) { return false; }
}

function isAuthError(res) {
    var m = res && res.message;
    return m === "Authentication required" || m === "Invalid credentials" || m === "Missing or malformed JWT" || m === "Invalid or expired JWT" || m === "Invalid or missing token" || m === "Token has been invalidated";
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
    return Response.error("Cần đăng nhập NovelHub: mở trình duyệt trong app, truy cập " + CONFIG_URL + "/login, đăng nhập rồi quay lại thử");
}

function execute(url) {
    // url is something like /api/v1/vbook/audio/playlist?book_id=xxx
    // Extract book_id
    var bookId = null;
    var match = url.match(/[?&]book_id=([^&]+)/);
    if (match) bookId = match[1];
    if (!bookId) {
        match = url.match(/[?&]id=([^&]+)/);
        if (match) bookId = match[1];
    }
    if (!bookId) {
        return Response.error("Không tìm thấy book_id");
    }
    var fetchUrl = CONFIG_URL + "/api/v1/vbook/detail?id=" + bookId;
    var res = fetchJson(fetchUrl);
    if (res && res.status && res.data) {
        var detail = res.data;
        if (detail.cover && detail.cover.indexOf("http") !== 0) {
            detail.cover = CONFIG_URL + (detail.cover.indexOf("/") === 0 ? "" : "/") + detail.cover;
        }
        detail.url = url;
        detail.type = "audio";
        detail.format = "album";
        return Response.success(detail);
    }
    if (res && res.needsLogin) return loginError();
    return Response.error("Không thể tải thông tin sách từ NovelHub");
}