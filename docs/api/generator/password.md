The Password generator provides random passwords that you can feed into your applications. It uses lower and uppercase alphanumeric characters as well as symbols. Please see below for the symbols in use.

!!! warning "Passwords are completely randomized"
    It is possible that we may generate passwords that don't match the expected character set from your application.

## Output Keys and Values

| Key      | Description            |
| -------- | ---------------------- |
| password | the generated password |

## Parameters

You can influence the behavior of the generator by providing the following args

| Key              | Default                            | Description                                                                 |
| ---------------- | ---------------------------------- | --------------------------------------------------------------------------- |
| length           | 24                                 | Length of the password to be generated.                                     |
| digits           | 25% of the length                  | Specify the number of digits in the generated password.                     |
| symbols          | 25% of the length                  | Specify the number of symbol characters in the generated.                   |
| symbolCharacters | ~!@#$%^&\*()\_+`-={}\|[]\\:"<>?,./ | Specify the character set that should be used when generating the password. |
| noUpper          | false                              | disable uppercase characters.                                               |
| allowRepeat      | false                              | allow repeating characters.                                                 |

## Example Manifest

```yaml
{% include 'generator-password.yaml' %}
```

Example `ExternalSecret` that references the Password generator:
```yaml
{% include 'generator-password-example.yaml' %}
```

Which will generate a `Kind=Secret` with a key called 'password' that may look like:

```
RMngCHKtZ@@h@3aja$WZDuDVhkCkN48JBa9OF8jH$R
VB$pX8SSUMIlk9K8g@XxJAhGz$0$ktbJ1ArMukg-bD
Hi$-aK_3Rrrw1Pj9-sIpPZuk5abvEDJlabUYUcS$9L
```

With default values you would get something like:

```
2Cp=O*&8x6sdwM!<74G_gUz5
-MS`e#n24K|h5A<&6q9Yv7Cj
ZRv-k!y6x/V"29:43aErSf$1
Vk9*mwXE30Q+>H?lY$5I64_q
```
