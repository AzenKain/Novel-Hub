function execute() {
    var response = Http.get("/api/v1/vbook/genres").string();
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            return Response.success(res.data);
        }
    }
    return Response.error("Không thể tải danh mục thể loại từ NovelHub");
}
