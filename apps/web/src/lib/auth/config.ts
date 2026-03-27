export type AuthMode = "mock" | "oidc";

export type AuthConfig = {
  mode: AuthMode;
  issuerUrl: string;
  clientId: string;
  redirectUri: string;
  authorizeUrl: string;
};

export function getAuthConfig(): AuthConfig {
  const rawMode = process.env.AUTH_MODE === "oidc" ? "oidc" : "mock";
  const issuerUrl = process.env.OIDC_ISSUER_URL ?? "";
  const authorizePath = process.env.OIDC_AUTHORIZE_PATH ?? "/authorize";
  const authorizeUrl = issuerUrl
    ? `${issuerUrl.replace(/\/$/, "")}${authorizePath.startsWith("/") ? authorizePath : `/${authorizePath}`}`
    : "";

  return {
    mode: rawMode,
    issuerUrl,
    clientId: process.env.OIDC_CLIENT_ID ?? "",
    redirectUri: process.env.OIDC_REDIRECT_URI ?? "http://localhost:3000/api/auth/callback",
    authorizeUrl,
  };
}
