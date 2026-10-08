---
appliesTo:
  - { type: capability, id: create-external-secret }
  - { type: capability, id: edit-external-secret }
references:
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/externalsecret_validator.go
---

# Every ExternalSecret fetches something

An ExternalSecret has at least one data entry or data-from source; each data-from source is exactly one of extract, find or a generator, and a source reference names a store or a generator. With deletion policy Retain, no two data entries write the same target key.
