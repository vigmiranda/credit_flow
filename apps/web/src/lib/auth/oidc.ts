import { createRemoteJWKSet, jwtVerify, type JWTPayload } from "jose";

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

type DiscoveryDocument = {
  issuer?: string;
  jwks_uri?: string;
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

export async function verifyIDToken(
  authConfig: AuthConfig,
  idToken: string,
): Promise<OidcProfile | null> {
  const metadata = await resolveDiscovery(authConfig);
  if (!metadata.jwksUri) {
    return decodeIDToken(idToken);
  }

  const jwks = createRemoteJWKSet(new URL(metadata.jwksUri));
  const verified = await jwtVerify(idToken, jwks, {
    issuer: metadata.issuer || authConfig.issuerUrl || undefined,
    audience: authConfig.clientId,
  });

  return payloadToProfile(verified.payload);
}

export function decodeIDToken(idToken: string): OidcProfile | null {
  const segments = idToken.split(".");
  if (segments.length < 2) {
    return null;
  }

  try {
    const payload = Buffer.from(toBase64(segments[1]), "base64").toString("utf-8");
    return payloadToProfile(JSON.parse(payload) as JWTPayload);
  } catch {
    return null;
  }
}

async function resolveDiscovery(authConfig: AuthConfig): Promise<{ issuer: string; jwksUri: string }> {
  if (authConfig.jwksUrl) {
    return {
      issuer: authConfig.issuerUrl,
      jwksUri: authConfig.jwksUrl,
    };
  }

  if (!authConfig.discoveryUrl) {
    return {
      issuer: authConfig.issuerUrl,
      jwksUri: "",
    };
  }

  const response = await fetch(authConfig.discoveryUrl, {
    method: "GET",
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error(`oidc_discovery_failed_${response.status}`);
  }

  const document = (await response.json()) as DiscoveryDocument;
  return {
    issuer: document.issuer || authConfig.issuerUrl,
    jwksUri: document.jwks_uri || "",
  };
}

function payloadToProfile(payload: JWTPayload): OidcProfile {
  return {
    sub: readClaim(payload, "sub"),
    name: readClaim(payload, "name"),
    email: readClaim(payload, "email"),
    role: readClaim(payload, "role") || readClaim(payload, "preferred_username"),
    preferred_username: readClaim(payload, "preferred_username"),
  };
}

function readClaim(payload: JWTPayload, claim: string): string {
  const value = payload[claim];
  return typeof value === "string" ? value : "";
}

function toBase64(value: string): string {
  return value.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(value.length / 4) * 4, "=");
}
