function execute(url, page) {
    if (!page) page = "1";
    var fetchUrl = url;
    if (fetchUrl.indexOf("?") >= 0) {
        fetchUrl += "&page=" + page;
    } else {
        fetchUrl += "?page=" + page;
    }

    var response = Http.get(fetchUrl).string();
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            return Response.success(res.data.list, res.data.next || null);
        }
    }
    return Response.error("Không thể tải danh sách sách từ NovelHub");
}
