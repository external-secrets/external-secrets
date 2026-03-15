# Incident response

This serves to define how potential security issues should be triaged, how
confirmation occurs, providing the notification, and issuing a security advisory
as well as patch/release.

## Triage

### Identify the problem

Triaging issues allows maintainers to focus resources on the most critically
impacting problems. Potential security risks should be evaluated against the
following information:

* Which component(s) of the project is impacted?
* What kind of problem is this?
  * privilege escalation
  * credential access
  * code execution
  * exfiltration
  * lateral movement
* How complex is the problem?
* Is user interaction required?
* What privileges are required for this problem to occur?
  * admin
  * general
* What is the potential impact or consequence of the problem?
* Does an exploit exist?

Any potential problem that has an exploit, permits privilege escalation, is
simple, and does not require user interaction should be evaluated immediately.
[CVSS Version 3.1](https://nvd.nist.gov/vuln-metrics/cvss/v3-calculator) can be
a helpful tool in evaluating the criticality of reported issues.

### Acknowledge receipt of the problem

Respond to the reporter and notify them that you have received and begun reviewing the problem. Remind them of the [embargo policy](https://github.com/cncf/tag-security/blob/231b87f371274b2d68def2c6a35a719210836191/project-resources/templates/embargo-policy.md), and provide them
information on who to contact/follow-up with if they have questions. Estimate when they can expect to receive an update. Create a calendar reminder to contact them again by that date to provide an update.

### Replicate the problem

Follow the instructions relayed in the problem. If the instructions are
insufficient, contact the reporter and ask for more information.

If the problem cannot be replicated, re-engage the reporter, let them know it
cannot be replicated, and work with them to find a remediation.

If the problem can be replicated, re-evaluate the criticality of the problem, and
begin working on a remediation. Begin a draft security advisory.

Notify the reporter you were able to replicate the problem and have begun working
on a fix. Remind them of the [embargo policy](https://github.com/cncf/tag-security/blob/231b87f371274b2d68def2c6a35a719210836191/project-resources/templates/embargo-policy.md). If necessary, notify them of an
extension (only for very complex problems where remediation cannot be issued
within the project's specified window).

#### Request a CVE number

If a CVE has already been provided, be sure to include it on the advisory. If
one has not yet been created, [GitHub functions as a
CNA](https://docs.github.com/en/code-security/security-advisories/about-github-security-advisories#cve-identification-numbers)
and allows you to request one as part of the security advisory process. Provide
all required information and as much optional information as we can. The CVE
number is shown as reserved with no further details until notified it has been
published.

## Notification

Once the problem has been replicated and a remediation is in place, notify
subscribed parties with a security bulletin (use [this template](https://github.com/cncf/tag-security/blob/231b87f371274b2d68def2c6a35a719210836191/project-resources/templates/embargo.md)) and the expected publishing date.

## Publish and release

Once a CVE number has been assigned, publish and release the updated
version/patch. Be sure to notify the CVE group when published so the CVE details
are searchable. Be sure to give credit to the reporter by *[editing the security
advisory](https://docs.github.com/en/github/managing-security-vulnerabilities/editing-a-security-advisory#about-credits-for-security-advisories)*
as they took the time to notify and work with you on the problem!

### Issue a security advisory

Follow the instructions from [GitHub to publish the security advisory previously
drafted](https://docs.github.com/en/github/managing-security-vulnerabilities/publishing-a-security-advisory).

For more information on security advisories, please refer to the [GitHub
Article](https://docs.github.com/en/code-security/security-advisories/about-github-security-advisories).