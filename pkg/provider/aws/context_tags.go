package aws

import esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"

// buildContextSessionTags returns additional STS session tags that encode
// Kubernetes execution context. The tags are only produced when `enabled` is
// true. Keys that are added:
//
//	esoNamespace     – namespace of the requesting resource
//	esoStoreName     – name of the SecretStore / ClusterSecretStore
//	esoOperatorRBAC  – (optional) name of the operator Role/ClusterRole
func buildContextSessionTags(enabled bool, namespace, storeName, operatorRBAC string) []*esv1.Tag {
	if !enabled {
		return nil
	}

	tags := []*esv1.Tag{
		&esv1.Tag{Key: "esoNamespace", Value: namespace},
		&esv1.Tag{Key: "esoStoreName", Value: storeName},
	}
	if operatorRBAC != "" {
		tags = append(tags, &esv1.Tag{Key: "esoOperatorRBAC", Value: operatorRBAC})
	}
	return tags
}
