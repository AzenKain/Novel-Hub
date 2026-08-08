function execute(keyword, page) {
    if (!page) page = "1";
    var fetchUrl = "/api/v1/vbook/search?q=" + encodeURIComponent(keyword) + "&page=" + page;

    var response = Http.get(fetchUrl).string();
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            return Response.success(res.data.list, res.data.next || null);
        }
    }
    return Response.error("Không tìm thấy kết quả từ NovelHub");
}
