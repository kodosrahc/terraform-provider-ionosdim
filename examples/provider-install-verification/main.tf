terraform {
  required_providers {
    ionosdim = {
      source = "ionos.com/aislab/ionosdim"
    }
  }
}

locals {
  my_l3domain = "default"
  my_zone     = "example.com"
}

provider "ionosdim" {
  endpoint = "https://dim-vrps.example.com/dim"
}

resource "ionosdim_ip" "ip_01" {
  layer3domain = local.my_l3domain
  cidr         = "10.88.8.0/22"
}

resource "ionosdim_a_record" "test01" {
  name         = "test01.${local.my_zone}."
  layer3domain = ionosdim_ip.ip_01.layer3domain
  ip           = ionosdim_ip.ip_01.ip
}

resource "ionosdim_cname_record" "test02" {
  name  = "test02.${local.my_zone}."
  cname = ionosdim_a_record.test01.name
}
