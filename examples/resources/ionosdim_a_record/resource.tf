# allocate an address to refer in record_01
resource "ionosdim_ip" "ip_01" {
  pool    = "some-pool"
  comment = "my comment"
}

resource "ionosdim_a_record" "record_01" {
  name         = "some-host.example.com."
  layer3domain = ionosdim_ip.ip_01.layer3domain
  ip           = ionosdim_ip.ip_01.ip
  comment      = "my comment"
}
