function execute() {
    var url = CONFIG_URL + "/api/v1/vbook/home";
    var response = Http.get(url).string();
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            return Response.success(res.data);
        }
    }
    return Response.error("Không thể tải danh sách Trang chủ từ NovelHub");
}

