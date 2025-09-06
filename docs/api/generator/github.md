## GitHub App Authentication Documentation

### 1. Register a GitHub App
To create a GitHub app, follow the instructions provided by GitHub:

- **Visit**: [Registering a GitHub App](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/registering-a-github-app#registering-a-github-app)
- **Procedure**:
  - Fill in the necessary details for your app.
  - Note the `App ID` provided after registration.
  - At the bottom of the registration page, click on `Generate a private key`. Download and securely store this key.

### 2. Store the Private Key
After generating your private key, you need to store it securely. If you are using Kubernetes, you can store it as a secret:

```bash
kubectl create secret generic github-app-pem --from-file=key=path/to/your/private-key.pem
```

### 3. Set Permissions for the GitHub App
Configure the necessary permissions for your GitHub app depending on what actions it needs to perform:

- **Visit**: [Choosing Permissions for a GitHub App](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/choosing-permissions-for-a-github-app#choosing-permissions-for-rest-api-access)
- **Example**:
  - For managing OCI images, set read and write permissions for packages.

### 4. Install Your GitHub App
Install the GitHub app on your repository or organization to start using it:

- **Visit**: [Installing Your Own GitHub App](https://docs.github.com/en/apps/using-github-apps/installing-your-own-github-app)

### 5. Obtain an Installation ID
After installation, you need to get the installation ID to authenticate API requests:

- **Visit**: [Generating an Installation Access Token for a GitHub App](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app#generating-an-installation-access-token)
- **Procedure**:
  - Find the installation ID from the URL or API response.

### Example Kubernetes Manifest for GitHub Access Token Generator

```yaml
{% include 'generator-github.yaml' %}
```

```yaml
{% include 'generator-github-example.yaml' %}
```

```yaml
{% include 'generator-github-example-basicauth.yaml' %}
```

### Notes
- Ensure that all sensitive data such as private keys and IDs are securely handled and stored.
- Adjust the permissions and configurations according to your specific requirements and security policies.
