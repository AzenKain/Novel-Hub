function execute(url) {
    var response = Http.get(url).string();
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            return Response.success(res.data.content);
        }
    }
    return Response.error("Không thể tải nội dung chương từ NovelHub");
}
