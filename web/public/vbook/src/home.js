function execute() {
    var url = CONFIG_URL + "/api/v1/vbook/home";
    var response = Http.get(url).string();
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            var sections = res.data;
            for (var i = 0; i < sections.length; i++) {
                if (sections[i].input && sections[i].input.indexOf("http") !== 0) {
                    sections[i].input = CONFIG_URL + (sections[i].input.indexOf("/") === 0 ? "" : "/") + sections[i].input;
                }
            }
            return Response.success(sections);
        }
    }
    return Response.error("Không thể tải danh sách Trang chủ từ NovelHub");
}

