import type { AuthConfig } from "@/lib/auth/config";

type TokenResponse = {
  access_token?: string;
  id_token?: string;
  token_type?: string;
  expires_in?: number;
};

type OidcProfile = {
  sub?: string;
  name?: string;
  email?: string;
  role?: string;
  preferred_username?: string;
};

export async function exchangeCodeForToken(
  authConfig: AuthConfig,
  code: string,
): Promise<TokenResponse> {
  const payload = new URLSearchParams({
    grant_type: "authorization_code",
    code,
    client_id: authConfig.clientId,
    redirect_uri: authConfig.redirectUri,
  });
  if (authConfig.clientSecret) {
    payload.set("client_secret", authConfig.clientSecret);
  }

  const response = await fetch(authConfig.tokenUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: payload.toString(),
    cache: "no-store",
  });

  if (!response.ok) {
    throw new Error(`oidc_token_exchange_failed_${response.status}`);
  }

  return (await response.json()) as TokenResponse;
}

export async function fetchUserInfo(
  authConfig: AuthConfig,
  accessToken: string,
): Promise<OidcProfile | null> {
  if (!authConfig.userInfoUrl || !accessToken) {
    return null;
  }

  const response = await fetch(authConfig.userInfoUrl, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error(`oidc_userinfo_failed_${response.status}`);
  }

  return (await response.json()) as OidcProfile;
}

export function decodeIDToken(idToken: string): OidcProfile | null {
  const segments = idToken.split(".");
  if (segments.length < 2) {
    return null;
  }

  try {
    const payload = Buffer.from(toBase64(segments[1]), "base64").toString("utf-8");
    return JSON.parse(payload) as OidcProfile;
  } catch {
    return null;
  }
}

function toBase64(value: string): string {
  return value.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(value.length / 4) * 4, "=");
}
