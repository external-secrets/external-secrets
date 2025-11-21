// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PostgreSQLCleanupPolicy controls the cleanup policy for the postgreSQL generator.
type PostgreSQLCleanupPolicy struct {
	CleanupPolicy `json:",inline"`

	// ActivityTrackingInterval is the cron expression to run the user activity tracking
	// +optional
	// +kubebuilder:default="2s"
	ActivityTrackingInterval metav1.Duration `json:"activityTrackingInterval,omitempty"`
}

// PostgreSQLSpec controls the behavior of the postgreSQL generator.
type PostgreSQLSpec struct {
	// Database is the name of the database to connect to.
	// If not specified, the "postgres" database will be used.
	// +kubebuilder:default=postgres
	Database string `json:"database"`
	// Host is the server where the database is hosted.
	Host string `json:"host"`
	// Port is the port of the database to connect to.
	// If not specified, the "5432" port will be used.
	// +kubebuilder:validation:Pattern=`^([0-9]{1,5}|[0-9]{1,5}\/[0-9]{1,5})$`
	// +kubebuilder:default="5432"
	Port string `json:"port"`
	// Auth contains the credentials or auth configuration
	Auth PostgreSQLAuth `json:"auth"`
	// User is the data of the user to be created.
	User *PostgreSQLUser `json:"user,omitempty"`

	CleanupPolicy *PostgreSQLCleanupPolicy `json:"cleanupPolicy,omitempty"`
}

// PostgreSQLAuth defines PostgreSQL authentication configuration.
type PostgreSQLAuth struct {
	// A basic auth username used to authenticate against the PostgreSQL instance.
	Username string `json:"username"`
	// A basic auth password used to authenticate against the PostgreSQL instance.
	Password esmeta.SecretKeySelector `json:"password"`
}

// PostgreSQLUserAttributesEnum represents PostgreSQL user attributes.
type PostgreSQLUserAttributesEnum string

const (
	// PostgreSQLUserSuperUser grants superuser privileges.
	PostgreSQLUserSuperUser PostgreSQLUserAttributesEnum = "SUPERUSER"
	// PostgreSQLUserCreateDb grants the ability to create databases.
	PostgreSQLUserCreateDb PostgreSQLUserAttributesEnum = "CREATEDB"
	// PostgreSQLUserCreateRole grants the ability to create roles.
	PostgreSQLUserCreateRole PostgreSQLUserAttributesEnum = "CREATEROLE"
	// PostgreSQLUserReplication grants the ability to replicate data.
	PostgreSQLUserReplication PostgreSQLUserAttributesEnum = "REPLICATION"
	// PostgreSQLUserNoInherit grants the ability to inherit privileges.
	PostgreSQLUserNoInherit PostgreSQLUserAttributesEnum = "NOINHERIT"
	// PostgreSQLUserByPassRls grants the ability to bypass row-level security.
	PostgreSQLUserByPassRls PostgreSQLUserAttributesEnum = "BYPASSRLS"
	// PostgreSQLUserConnectionLimit grants the ability to limit the number of connections.
	PostgreSQLUserConnectionLimit PostgreSQLUserAttributesEnum = "CONNECTION LIMIT"
	// PostgreSQLUserLogin grants the ability to login.
	PostgreSQLUserLogin PostgreSQLUserAttributesEnum = "LOGIN"
	// PostgreSQLUserPassword grants the ability to set a password.
	PostgreSQLUserPassword PostgreSQLUserAttributesEnum = "PASSWORD"
)

// PostgreSQLUser defines a PostgreSQL user.
type PostgreSQLUser struct {
	// The username of the user to be created.
	Username string `json:"username"`
	// SuffixSize define the size of the random suffix added after the defined username.
	// If not specified, a random suffix of size 8 will be used.
	// If set to 0, no suffix will be added.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=8
	SuffixSize *int `json:"suffixSize,omitempty"`
	// Attributes is the list of PostgreSQL role attributes assigned to this user.
	Attributes []PostgreSQLUserAttribute `json:"attributes,omitempty"`
	// Roles is the list of existing roles that will be granted to this user.
	// If a role does not exist, it will be created without any attributes.
	Roles []string `json:"roles,omitempty"`
	// If set to true, the generator will drop all objects owned by the user
	// before deleting the user during cleanup.
	// If false (default), ownership of all objects will be reassigned
	// to the role specified in `spec.user.reassignTo`.
	// +kubebuilder:default=false
	DestructiveCleanup bool `json:"destructiveCleanup,omitempty"`
	// The name of the role to which all owned objects should be reassigned
	// during cleanup (if DestructiveCleanup is false).
	// If not specified, the role from `spec.auth.username` will be used.
	// If the role does not exist, it will be created with no attributes or roles..
	ReassignTo *string `json:"reassignTo,omitempty"`
}

// PostgreSQLUserAttribute defines a PostgreSQL user attribute.
type PostgreSQLUserAttribute struct {
	// Attribute is the name of the PostgreSQL role attribute to be set for the user.
	// Valid values: SUPERUSER, CREATEDB, CREATEROLE, REPLICATION, NOINHERIT, BYPASSRLS, CONNECTION_LIMIT.
	// +kubebuilder:validation:Enum=SUPERUSER;CREATEDB;CREATEROLE;REPLICATION;NOINHERIT;BYPASSRLS;CONNECTION_LIMIT
	Name string `json:"name"`
	// Optional value for the attribute (e.g., connection limit)
	Value *string `json:"value,omitempty"`
}

// PostgreSQLUserState represents the state of a PostgreSQL user.
type PostgreSQLUserState struct {
	Username string `json:"username,omitempty"`
}

// PostgreSQL generates a PostgreSQL user based on the configuration parameters in spec.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type PostgreSQL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgreSQLSpec  `json:"spec,omitempty"`
	Status GeneratorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PostgreSQLList contains a list of PostgreSQL resources.
type PostgreSQLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSQL `json:"items"`
}
