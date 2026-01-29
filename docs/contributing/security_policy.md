<!-- Reference. We explain WHAT is the content of our policy, verbatim -->
<!-- If you want to contribute to this page: -->
<!-- If you are looking for "HOW DO I follow ESO process to report security issue", look for the SECURITY.md page -->
<!-- If you are looking for "HOW DO I follow ESI process to work on a security issue", look for SECURITY_RESPONSE page) -->
<!-- If you are looking for WHY we have security process x, look for our general policy -->
## Vulnerability Management
### Responsible disclosure policy

ESO follows responsible disclosure practices.

Security-impacting issues should be reported via our documented security contact channels (see `SECURITY.md`).
Security fixes may be handled privately until a coordinated disclosure and release are ready (see `SECURITY_RESPONSE.md`)

### Critical Updates And Security Notices

On top of responsible disclosures about our software, we learn about critical software updates/security threats from:

1. GitHub Security Alerts
2. [Dependabot](https://dependabot.com/) Dependency Updates

The community regularly fixes security issues and produce releases.
There is no SLA on issues. You use this software at your own risk.

## Helm Chart and "Security by default"

Our Helm charts are designed for ease of use and general-purpose scenarios.
We strongly recommend that you review the default configuration and harden it to fit your security requirements.

You can do this by customizing the chart values, or by using our chart as a dependency and extending it with your own security measures,
such as NetworkPolicies, Admission Control logic, or other controls.

Any misconfiguration caused by using the provided helm charts is not covered by our policy - even if it leads to a security incident.


