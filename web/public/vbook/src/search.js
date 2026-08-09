function execute(keyword, page) {
    if (!page) page = "1";
    var fetchUrl = CONFIG_URL + "/api/v1/vbook/search?q=" + encodeURIComponent(keyword) + "&page=" + page;

    var response = Http.get(fetchUrl).string();
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            var list = res.data.list || [];
            for (var i = 0; i < list.length; i++) {
                if (list[i].cover && list[i].cover.indexOf("http") !== 0) {
                    list[i].cover = CONFIG_URL + (list[i].cover.indexOf("/") === 0 ? "" : "/") + list[i].cover;
                }
                if (list[i].link && list[i].link.indexOf("http") !== 0) {
                    list[i].link = CONFIG_URL + (list[i].link.indexOf("/") === 0 ? "" : "/") + list[i].link;
                }
            }
            return Response.success(list, res.data.next || null);
        }
    }
    return Response.error("Không tìm thấy kết quả từ NovelHub");
}

