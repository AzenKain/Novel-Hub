var CONFIG_URL = "{{BASE_URL}}";

function execute() {
    return Response.success([
        {
            title: "Audiobook mới cập nhật",
            input: CONFIG_URL + "/api/v1/vbook/audio/books",
            script: "gen.js"
        }
    ]);
}