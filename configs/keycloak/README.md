# Keycloak Configuration

This directory contains Keycloak realm configuration for the Flowra application.

## Files

- `realm-export.json` - Complete realm configuration including:
  - Realm settings (login, brute force protection)
  - OAuth2 client `flowra-backend`
  - Realm roles (user, admin, workspace_owner, workspace_admin)
  - Default groups (users, admins)
  - Client scopes with protocol mappers
  - Test users for development

## Auto-Import on Docker Compose

The Docker Compose configuration mounts this directory and imports the realm automatically on startup:

```yaml
keycloak:
  image: quay.io/keycloak/keycloak:23.0
  volumes:
    - ./configs/keycloak:/opt/keycloak/data/import
  command: start-dev --import-realm
```

Simply run `docker-compose up -d` and the realm will be imported automatically.

## Manual Import

If you need to import manually:

### Via Keycloak Admin Console

1. Open http://localhost:8090/admin
2. Login with admin/admin123
3. Click "Create Realm"
4. Toggle "Resource file" and upload `realm-export.json`
5. Click "Create"

### Via Admin REST API

```bash
# Get admin token
ADMIN_TOKEN=$(curl -s -X POST "http://localhost:8090/realms/master/protocol/openid-connect/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "username=admin" \
    -d "password=admin123" \
    -d "grant_type=password" \
    -d "client_id=admin-cli" | jq -r '.access_token')

# Import realm
curl -X POST "http://localhost:8090/admin/realms" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d @configs/keycloak/realm-export.json
```

## Test Users

| Username   | Email                  | Password     | Roles                  |
|------------|------------------------|--------------|------------------------|
| testuser   | testuser@example.com   | password123  | user                   |
| admin      | admin@example.com      | admin123     | user, admin            |
| alice      | alice@example.com      | password123  | user, workspace_owner  |
| bob        | bob@example.com        | password123  | user                   |

## OAuth2 Client Configuration

| Setting              | Value                                          |
|----------------------|------------------------------------------------|
| Client ID            | flowra-backend                                 |
| Client Secret        | flowra-dev-secret-change-in-production         |
| Redirect URIs        | http://localhost:8080/auth/callback, http://localhost:3000/auth/callback |
| Web Origins          | http://localhost:8080, http://localhost:3000   |
| Standard Flow        | Enabled                                        |
| Direct Access Grants | Enabled (for testing)                          |

## JWT Token Claims

After authentication, tokens will contain:

```json
{
  "iss": "http://localhost:8090/realms/flowra",
  "aud": "flowra-backend",
  "sub": "user-uuid",
  "scope": "openid profile email",
  "email_verified": true,
  "name": "Test User",
  "preferred_username": "testuser",
  "given_name": "Test",
  "family_name": "User",
  "email": "testuser@example.com",
  "realm_access": {
    "roles": ["user", "default-roles-flowra"]
  },
  "groups": ["/users"]
}
```

## Production Notes

⚠️ **Do NOT use this configuration in production!**

Before deploying to production:

1. Generate a new client secret
2. Remove test users or change passwords
3. Configure proper redirect URIs
4. Enable HTTPS
5. Configure email settings for password reset
6. Review brute force protection settings
7. Enable email verification if needed

## Troubleshooting

### Realm already exists

If you see "Realm already exists" error, the realm was previously imported. You can:

1. Delete the realm via Admin Console and restart Keycloak
2. Or use `--override` flag (Keycloak 24+)

### Token validation fails

Ensure the client secret in your application config matches the one in `realm-export.json`:

```yaml
keycloak:
  client_secret: "flowra-dev-secret-change-in-production"
```

### Groups not in token

Add the `groups` scope to your token request:

```
scope=openid profile email groups
```
