---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "ionosdim_ip Resource - terraform-provider-ionosdim"
subcategory: ""
description: |-
  Allocates an ip address from the pool (i.e. set status to Static).
   - If the ip argument left unspecified, it will allocate the next free (status = Available) ip address from the pool;
   - If ip is specified, it must be free (status = Available ) upon resource creation.
---

# ionosdim_ip (Resource)

Allocates an ip address from the pool (i.e. set `status` to `Static`).
 - If the `ip` argument left unspecified, it will allocate the next free (`status` = `Available`) ip address from the pool;
 - If `ip` is specified, it must be free (`status` = `Available` ) upon resource creation.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `pool` (String) The pool where the IP address is allocated.

### Optional

- `comment` (String) The comment to the allocated IP address
- `ip` (String) If specified, this address will be allocated. The address must be within the `pool` specified. If not set, an available address will be allocated from the pool.

### Read-Only

- `created` (String)
- `gateway` (String)
- `id` (String) The ID of this resource.
- `layer3domain` (String) The layer 3 domain where the IP address is allocated.
- `mask` (String)
- `modified` (String)
- `modified_by` (String)
- `reverse_zone` (String)
- `status` (String) the known status values are:
  - `Static` a single allocated IP address
  - `Available` a single free IP address
  - `Reserved` a reserved single IP address (for example the IPv4 network and broadcast addresses in a Subnet)
  - `Container` a generic status for blocks larger than subnets
  - `Delegation` the block is used for a specific purpose (ex: a server)
  - `Subnet` a subnet (can only have Delegation, Static, Reserved or Available children)
- `subnet` (String)
