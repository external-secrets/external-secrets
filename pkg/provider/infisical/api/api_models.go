package api

type MachineIdentityUniversalAuthRefreshRequest struct {
	AccessToken string `json:"accessToken"`
}

type MachineIdentityDetailsResponse struct {
	AccessToken       string `json:"accessToken"`
	ExpiresIn         int    `json:"expiresIn"`
	AccessTokenMaxTTL int    `json:"accessTokenMaxTTL"`
	TokenType         string `json:"tokenType"`
}

type MachineIdentityUniversalAuthLoginRequest struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type GetSecretByKeyV3Request struct {
	EnvironmentSlug string `json:"environment"`
	ProjectSlug     string `json:"workspaceSlug"`
	SecretPath      string `json:"secretPath"`
	SecretKey       string `json:"secretKey"`
}

type GetSecretByKeyV3Response struct {
	Secret SecretsV3 `json:"secret"`
}

type GetSecretsV3Request struct {
	EnvironmentSlug string `json:"environment"`
	ProjectSlug     string `json:"workspaceSlug"`
	SecretPath      string `json:"secretPath"`
}

type GetSecretsV3Response struct {
	Secrets         []SecretsV3        `json:"secrets"`
	ImportedSecrets []ImportedSecretV3 `json:"imports,omitempty"`
	Modified        bool               `json:"modified,omitempty"`
	ETag            string             `json:"ETag,omitempty"`
}

type SecretsV3 struct {
	ID            string `json:"id"`
	Workspace     string `json:"workspace"`
	Environment   string `json:"environment"`
	Version       int    `json:"version"`
	Type          string `json:"string"`
	SecretKey     string `json:"secretKey"`
	SecretValue   string `json:"secretValue"`
	SecretComment string `json:"secretComment"`
}

type ImportedSecretV3 struct {
	Environment string      `json:"environment"`
	FolderId    string      `json:"folderId"`
	SecretPath  string      `json:"secretPath"`
	Secrets     []SecretsV3 `json:"secrets"`
}

type InfisicalApiErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Error      any    `json:"error"`
}
