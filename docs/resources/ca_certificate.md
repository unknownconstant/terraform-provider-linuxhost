---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "linuxhost_ca_certificate Resource - linuxhost"
subcategory: ""
description: |-
  A trusted root certificate on the host
---

# linuxhost_ca_certificate (Resource)

A trusted root certificate on the host



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Human-readable name for the certificate, also used as its filename
- `source` (String) The certificate source location. For a file, a standard unix path. Or, https://example.com/certificate.pem.

### Read-Only

- `fingerprint_sha256` (String)
