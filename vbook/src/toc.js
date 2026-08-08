function execute(url) {
    var tocUrl = url;
    if (tocUrl.indexOf("/detail?") >= 0) {
        tocUrl = tocUrl.replace("/detail?", "/toc?");
    } else if (tocUrl.indexOf("/api/v1/vbook/toc") < 0) {
        var matches = url.match(/[?&]id=([^&]+)/);
        if (matches && matches[1]) {
            tocUrl = "/api/v1/vbook/toc?id=" + matches[1];
        }
    }

    var response = Http.get(tocUrl).string();
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            return Response.success(res.data);
        }
    }
    return Response.error("Không thể tải mục lục từ NovelHub");
}
