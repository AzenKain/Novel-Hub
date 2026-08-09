var CONFIG_URL = "{{BASE_URL}}";
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

    if (tocUrl.indexOf("http") !== 0) {
        tocUrl = CONFIG_URL + (tocUrl.indexOf("/") === 0 ? "" : "/") + tocUrl;
    }

    var response = Http.get(tocUrl).string("UTF-8");
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            var list = res.data || [];
            for (var i = 0; i < list.length; i++) {
                if (list[i].url && list[i].url.indexOf("http") !== 0) {
                    list[i].url = CONFIG_URL + (list[i].url.indexOf("/") === 0 ? "" : "/") + list[i].url;
                }
            }
            return Response.success(list);
        }
    }
    return Response.error("Không thể tải mục lục từ NovelHub");
}

