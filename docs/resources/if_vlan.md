---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "linuxhost_if_vlan Resource - linuxhost"
subcategory: ""
description: |-
  A vlan interface
---

# linuxhost_if_vlan (Resource)

A vlan interface



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The interface identifier, .e.g. 'eth0'
- `parent` (String) The parent interface, e.g. eth0
- `vid` (Number) The VLAN ID

### Optional

- `bridge` (Attributes) If specified, the bridge this interface is a member of. (see [below for nested schema](#nestedatt--bridge))
- `state` (String) Interface state. Valid options: 'up', 'down'.

### Read-Only

- `ipv4` (Set of String)
- `mac` (String) The assigned interface mac address

<a id="nestedatt--bridge"></a>
### Nested Schema for `bridge`

Required:

- `name` (String) The name of the bridge, e.g. 'br0'
