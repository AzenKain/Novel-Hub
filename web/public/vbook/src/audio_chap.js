var CONFIG_URL = "{{BASE_URL}}";

function execute(url) {
  // url is already the stream URL, just pass it through
  var data = url;
  if (data.indexOf("http") !== 0) {
    data = CONFIG_URL + (data.indexOf("/") === 0 ? "" : "/") + data;
  }
  return Response.success([{ title: "", data: data }]);
}
