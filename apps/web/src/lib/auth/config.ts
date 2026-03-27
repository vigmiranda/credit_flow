export type AuthMode = "mock" | "oidc";

export type AuthConfig = {
  mode: AuthMode;
  issuerUrl: string;
  clientId: string;
  redirectUri: string;
};

export function getAuthConfig(): AuthConfig {
  const rawMode = process.env.AUTH_MODE === "oidc" ? "oidc" : "mock";

  return {
    mode: rawMode,
    issuerUrl: process.env.OIDC_ISSUER_URL ?? "",
    clientId: process.env.OIDC_CLIENT_ID ?? "",
    redirectUri: process.env.OIDC_REDIRECT_URI ?? "http://localhost:3000/api/auth/callback",
  };
}
