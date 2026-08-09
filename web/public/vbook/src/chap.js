var CONFIG_URL = "{{BASE_URL}}";
function execute(url) {
    var fetchUrl = url;
    if (fetchUrl.indexOf("http") !== 0) {
        fetchUrl = CONFIG_URL + (fetchUrl.indexOf("/") === 0 ? "" : "/") + fetchUrl;
    }
    var response = Http.get(fetchUrl).string("UTF-8");
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            return Response.success(makeAbsolute(res.data.content));
        }
    }
    return Response.error("Không thể tải nội dung chương từ NovelHub");
}

function makeAbsolute(html) {
    if (!html) return html;
    return html.replace(/src\s*=\s*"([^"]+)"/g, function (m, url) {
        if (url.indexOf("http") === 0 || url.indexOf("//") === 0 || url.indexOf("data:") === 0 || url.charAt(0) === "#") {
            return m;
        }
        return m.replace('"' + url + '"', '"' + CONFIG_URL + (url.charAt(0) === "/" ? "" : "/") + url + '"');
    });
}

