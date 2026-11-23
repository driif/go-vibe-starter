# Authentication

The API is an OAuth resource server. HTTP middleware verifies bearer access tokens before decoding
RPC bodies, then stores an `identity.Principal` in the request context. `identity.FromContext(ctx)`
returns it. Role checks belong beside the operation they protect:

```go
if !identity.FromContext(ctx).HasRole("admin") {
    return nil, apperr.New(apperr.PermissionDenied, "ADMIN_REQUIRED", "admin role required")
}
```

Invalid/missing credentials return `unauthenticated` (401), provider transport failures
`unavailable` (503), and rejected roles `permission_denied` (403). Debug logging includes rejection
reasons without tokens. `AUTH_PROVIDER=none` is explicitly local-only and grants no built-in roles; the local env file explicitly adds admin.

## Local Keycloak

Run `make keycloak-up`. In `.env.local`, set `AUTH_PROVIDER=keycloak`; the supplied issuer is
`http://localhost:8181/realms/app` and audience is `api`. Use `localhost`, not `127.0.0.1`, and no
trailing slash: the configured URL must equal the token issuer exactly. `make -s token` obtains an
access token for alice; `make -s token KC_USER=bob` gets a user-only token. `make smoke` exercises
the shipped realm. The password grant and all fixture credentials are local test tools.

## Production Keycloak

Create a separate realm; do not deploy the dev realm. Use HTTPS and an access-token audience mapper
for the API on every calling client. Keep the API audience distinct from client ids and out of ID
tokens. Disable direct access grants. Use short access-token lifetimes and test key rotation.
After deleting a signing key, restart API replicas to discard previously cached keys.
## Directory login through Keycloak

1. In the deployment realm, open **User federation → Add LDAP provider**. Select the directory vendor and set the users DN, the LDAPS URL, and a read-only bind account. Store the bind credential in Keycloak's secret store. Configure trust for the directory's CA.
2. Test the connection and authentication in Keycloak. Choose read-only edit mode when the directory owns user records. Configure LDAP group mappings and assign the realm roles your API uses.
3. Sign in through the browser client as a directory user. Call WhoAmI with that user's access token and check the subject and roles. The Go service uses the same issuer and API audience for local and federated accounts.

See the [Keycloak server administration guide](https://www.keycloak.org/docs/latest/server_admin/#_ldap) for vendor settings, mappers and synchronization.

## Verifier limits

Tokens are checked locally against discovery/JWKS, including issuer, audience, expiry, signature,
nonempty subject and Keycloak access-token type. Readiness requires keys to have loaded once; a
later provider outage does not remove readiness. Previously cached keys can verify tokens during
an outage. Unknown keys may yield unavailable when refresh fails. A short successful-JWKS replay
window limits repeated refreshes from forged key ids and delays recognition of a newly rotated key.

There is no token introspection or immediate logout/revocation. A valid token remains usable until
expiry; deleted keys may remain cached. Auth runs once at the start of a stream. Set application
stream limits where needed; see [production guidance](production.md).

## Browser applications

The frontend is created separately under `frontend/`. Use Authorization Code with PKCE and exact
redirect URIs. A backend-for-frontend with secure HttpOnly cookies is appropriate when the browser
must not hold tokens; it is a separate application/session boundary. This API ships no login route,
cookie session, refresh-token store, CSRF flow or frontend. Allow exact frontend origins with
`CORS_ALLOWED_ORIGINS`; bearer tokens travel in headers, never URLs.

## Service applications

Use client credentials from confidential clients and supply the resulting bearer token. Keep secrets
in the deployment's secret store. The app needs only issuer and audience, never a realm admin secret.
There is no Keycloak admin client, direct LDAP bind, tenancy or general authorization framework.
