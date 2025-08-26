# Security Policy

- [Security Policy](#security-policy)
  - [Reporting security problems](#reporting-security-problems)
  - [Vulnerability Management Plans](#vulnerability-management-plans)
    - [Critical Updates And Security Notices](#critical-updates-and-security-notices)

<a name="reporting"></a>
## Reporting security problems

**DO NOT CREATE AN ISSUE** to report a security problem. Instead, please
send an email to cncf-ExternalSecretsOp-maintainers@lists.cncf.io

<a name="vulnerability-management"></a>
## Vulnerability Management Plans

### Critical Updates And Security Notices

We learn about critical software updates and security threats from these sources

1. GitHub Security Alerts
2. [Dependabot](https://dependabot.com/) Dependency Updates

## Helm Chart Security

Our Helm charts are designed for ease of use and general-purpose scenarios. We strongly recommend that you review the default configuration and harden it to fit your security requirements. 

You can do this by customizing the chart values, or by using our chart as a dependency and extending it with your own security measures, such as NetworkPolicies, Admission Control logic, or other controls.

Any misconfiguration caused by using the provided helm charts is not covered by our support policy - even if it leads to a security incident.

## Security Incident Response

Please follow the guide [SECURITY_RESPONSE.md](SECURITY_RESPONSE.md).
