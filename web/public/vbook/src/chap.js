function execute(url) {
    var fetchUrl = url;
    if (fetchUrl.indexOf("http") !== 0) {
        fetchUrl = CONFIG_URL + (fetchUrl.indexOf("/") === 0 ? "" : "/") + fetchUrl;
    }
    var response = Http.get(fetchUrl).string();
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            return Response.success(res.data.content);
        }
    }
    return Response.error("Không thể tải nội dung chương từ NovelHub");
}

