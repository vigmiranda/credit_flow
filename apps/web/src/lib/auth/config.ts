export type AuthMode = "mock" | "oidc";

export type AuthConfig = {
  mode: AuthMode;
  issuerUrl: string;
  clientId: string;
  clientSecret: string;
  redirectUri: string;
  discoveryUrl: string;
  authorizeUrl: string;
  tokenUrl: string;
  userInfoUrl: string;
  jwksUrl: string;
};

export function getAuthConfig(): AuthConfig {
  const rawMode = process.env.AUTH_MODE === "oidc" ? "oidc" : "mock";
  const issuerUrl = process.env.OIDC_ISSUER_URL ?? "";
  const discoveryUrl = process.env.OIDC_DISCOVERY_URL
    ? process.env.OIDC_DISCOVERY_URL
    : issuerUrl
      ? `${issuerUrl.replace(/\/$/, "")}/.well-known/openid-configuration`
      : "";
  const authorizePath = process.env.OIDC_AUTHORIZE_PATH ?? "/authorize";
  const tokenPath = process.env.OIDC_TOKEN_PATH ?? "/token";
  const userInfoPath = process.env.OIDC_USERINFO_PATH ?? "/userinfo";
  const authorizeUrl = issuerUrl
    ? `${issuerUrl.replace(/\/$/, "")}${authorizePath.startsWith("/") ? authorizePath : `/${authorizePath}`}`
    : "";
  const tokenUrl = process.env.OIDC_TOKEN_URL
    ? process.env.OIDC_TOKEN_URL
    : issuerUrl
      ? `${issuerUrl.replace(/\/$/, "")}${tokenPath.startsWith("/") ? tokenPath : `/${tokenPath}`}`
      : "";
  const userInfoUrl = process.env.OIDC_USERINFO_URL
    ? process.env.OIDC_USERINFO_URL
    : issuerUrl
      ? `${issuerUrl.replace(/\/$/, "")}${userInfoPath.startsWith("/") ? userInfoPath : `/${userInfoPath}`}`
      : "";
  const jwksUrl = process.env.OIDC_JWKS_URL ?? "";

  return {
    mode: rawMode,
    issuerUrl,
    clientId: process.env.OIDC_CLIENT_ID ?? "",
    clientSecret: process.env.OIDC_CLIENT_SECRET ?? "",
    redirectUri: process.env.OIDC_REDIRECT_URI ?? "http://localhost:3000/api/auth/callback",
    discoveryUrl,
    authorizeUrl,
    tokenUrl,
    userInfoUrl,
    jwksUrl,
  };
}
