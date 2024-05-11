terraform {
  required_providers {
    ionosdim = {
      source  = "kodosrahc/ionosdim"
      version = ">= 0.1.2"
    }
  }
}

locals {
  my_zone      = "example.com"
  my_pool      = "some-pool"
  my_host_name = "some-host"
  my_canonical = "${local.my_host_name}.${local.my_zone}."
}

provider "ionosdim" {
  endpoint = "https://dim.example.com/dim"
}


resource "ionosdim_ip" "ip_01" {
  pool    = local.my_pool
  comment = "allocating some available address"
}

resource "ionosdim_ip" "ip_02" {
  pool    = local.my_pool
  comment = "allocating the specific address"
  ip      = "10.88.8.18"
}

resource "ionosdim_a_record" "ip_01_a_record" {
  name         = local.my_canonical
  zone         = local.my_zone # optional when `name` is FQDN with a trailing dot
  layer3domain = ionosdim_ip.ip_01.layer3domain
  ip           = ionosdim_ip.ip_01.ip
  comment      = ionosdim_ip.ip_01.comment
}

resource "ionosdim_cname_record" "ip_01_catchall_cname_record" {
  name    = "*.${local.my_host_name}"
  zone    = local.my_zone # needed when `name` is not FQDN
  comment = ionosdim_ip.ip_01.comment
  cname   = local.my_canonical
}
