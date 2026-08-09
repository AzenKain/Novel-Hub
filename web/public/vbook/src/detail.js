function execute(url) {
    var fetchUrl = url;
    if (fetchUrl.indexOf("http") !== 0) {
        fetchUrl = CONFIG_URL + (fetchUrl.indexOf("/") === 0 ? "" : "/") + fetchUrl;
    }
    var response = Http.get(fetchUrl).string();
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            var detail = res.data;
            if (detail.cover && detail.cover.indexOf("http") !== 0) {
                detail.cover = CONFIG_URL + (detail.cover.indexOf("/") === 0 ? "" : "/") + detail.cover;
            }
            return Response.success(detail);
        }
    }
    return Response.error("Không thể tải thông tin sách từ NovelHub");
}

