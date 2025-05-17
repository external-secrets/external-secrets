# MFA Generator

This generator can create [RFC 4226](https://datatracker.ietf.org/doc/html/rfc4226) compliant TOTP tokens given a seed
secret. The seed secret is usually provided through a QR code. However, the provider will always also provide a text
based format of that QR code. That's the secret that this generator will use to create tokens.

## Output Keys and Values

| Key      | Description                                      |
|----------|--------------------------------------------------|
| token    | the generated N letter token                     |
| timeLeft | the time left until the token expires in seconds |

## Parameters

The following configuration options are available when generating a token:

| Key        | Default  | Description                                                                                                    |
|------------|----------|----------------------------------------------------------------------------------------------------------------|
| length     | 6        | Digit length of the generated code. Some providers allow larger tokens.                                        |
| timePeriod | 30       | Number of seconds the code can be valid. This is provider specific, usually it's 30 seconds                    |
| secret     | empty    | This is a secret ref pointing to the seed secret                                                               |
| algorithm  | sha1     | Algorithm for encoding. The RFC defines SHA1, though a provider will set it to SHA256 or SHA512 sometimes      |
| when       | time.Now | This allows for pinning the creation date of the token makes for reproducible tokens. Mostly used for testing. |

## Example Manifest

```yaml
{% include 'generator-mfa.yaml' %}
```

This will generate an output like this:

```
token: 123456
timeLeft: 25
```

!!! warning "Usage of the token might fail on first try if it JUST expired"
It is possible that from requesting the token to actually using it, the token might be already out of date if timeLeft was
very low to begin with. Therefor, the code that uses this token should allow for retries with new tokens.
