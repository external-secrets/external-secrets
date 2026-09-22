// Vault buildMetadataPath: strip mount prefix to avoid incorrect path resolution.
// This ensures the resolved secret path excludes the Vault mount prefix for consistent lookup.
