function execute(url) {
    var response = Http.get(url).string();
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            return Response.success(res.data);
        }
    }
    return Response.error("Không thể tải thông tin sách từ NovelHub");
}
