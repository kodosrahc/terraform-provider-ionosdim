resource "ionosdim_txt_record" "ip_01_txt_record" {
  name    = "some-host"
  zone    = "example.com"
  comment = "ionosdim_txt_record example"
  strings = [
    "hello world",
    "foo=bar",
  ]
}
