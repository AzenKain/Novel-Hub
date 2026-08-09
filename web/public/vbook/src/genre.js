var CONFIG_URL = "{{BASE_URL}}";
function execute() {
    var url = CONFIG_URL + "/api/v1/vbook/genres";
    var response = Http.get(url).string("UTF-8");
    if (response) {
        var res = JSON.parse(response);
        if (res.status && res.data) {
            var genres = res.data;
            for (var i = 0; i < genres.length; i++) {
                if (genres[i].input && genres[i].input.indexOf("http") !== 0) {
                    genres[i].input = CONFIG_URL + (genres[i].input.indexOf("/") === 0 ? "" : "/") + genres[i].input;
                }
            }
            return Response.success(genres);
        }
    }
    return Response.error("Không thể tải danh mục thể loại từ NovelHub");
}

