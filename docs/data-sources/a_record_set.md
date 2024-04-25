---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "ionosdim_a_record_set Data Source - terraform-provider-ionosdim"
subcategory: ""
description: |-
  Use this data source to get DNS A records of the host.
---

# ionosdim_a_record_set (Data Source)

Use this data source to get DNS A records of the host.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `host` (String) Host to look up.
- `zone` (String) Zone.

### Optional

- `layer3domain` (String) layer3domain.
- `view` (String) View.

### Read-Only

- `addrs` (List of String) A list of IP addresses. IP addresses are always sorted to avoid constant changing plans.
- `id` (String) Always set to the host.
- `ttl` (Number) ttl.