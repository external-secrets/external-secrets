# esoctl

This tool contains external-secrets-operator related activities and helpers.

## Templates

`cmd/render` -> `esoctl template`

The purpose is to give users the ability to rapidly test and iterate on templates in a PushSecret/ExternalSecret.

For a more in-dept description read [Using Render Tool](../../docs/guides/using-render-tool.md).

This project doesn't have its own go mod files to allow it to grow together with ESO instead of waiting for new ESO
releases to import it.
