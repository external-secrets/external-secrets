---
entities:
  - entity: push-secret
    collects: [Secret stores, Source, Data, Data to, Template, Update policy, Deletion policy, Refresh interval]
    shows: [Ready, Synced push secrets, Refresh time]
---

# PushSecret

One PushSecret in a namespace: what it pushes to which stores, with the outcome
of its last push and the provider keys it has written.
