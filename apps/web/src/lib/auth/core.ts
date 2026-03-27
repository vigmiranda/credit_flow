export const AUTH_COOKIE_NAME = "credit_flow_session";

export type AuthSession = {
  userId: string;
  name: string;
  email: string;
  role: string;
  authMode: "mock" | "oidc";
};

export function serializeSession(session: AuthSession): string {
  return encodeURIComponent(JSON.stringify(session));
}

export function parseSession(value?: string | null): AuthSession | null {
  if (!value) {
    return null;
  }

  try {
    const parsed = JSON.parse(decodeURIComponent(value)) as Partial<AuthSession>;
    if (!parsed.userId || !parsed.name || !parsed.email || !parsed.role) {
      return null;
    }

    return {
      userId: parsed.userId,
      name: parsed.name,
      email: parsed.email,
      role: parsed.role,
      authMode: parsed.authMode === "oidc" ? "oidc" : "mock",
    };
  } catch {
    return null;
  }
}
