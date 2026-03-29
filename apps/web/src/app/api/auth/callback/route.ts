import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { getAuthConfig } from "@/lib/auth/config";
import { AUTH_COOKIE_NAME, serializeSession } from "@/lib/auth/core";
import { exchangeCodeForToken, fetchUserInfo, verifyIDToken } from "@/lib/auth/oidc";

const OIDC_STATE_COOKIE = "credit_flow_oidc_state";

export async function GET(request: Request) {
  const authConfig = getAuthConfig();
  const url = new URL(request.url);
  const code = url.searchParams.get("code")?.trim();
  const state = url.searchParams.get("state")?.trim();
  const authError = url.searchParams.get("error")?.trim();

  if (authConfig.mode !== "oidc") {
    return NextResponse.redirect(new URL("/login", request.url), 303);
  }

  if (authError) {
    return NextResponse.redirect(
      new URL(`/login?error=${encodeURIComponent(authError)}`, request.url),
      303,
    );
  }

  const cookieStore = await cookies();
  const expectedState = cookieStore.get(OIDC_STATE_COOKIE)?.value;
  if (!code || !state || !expectedState || state !== expectedState) {
    return NextResponse.redirect(new URL("/login?error=invalid_oidc_callback", request.url), 303);
  }

  let profile = {
    sub: url.searchParams.get("sub")?.trim() || "",
    name: url.searchParams.get("name")?.trim() || "",
    email: url.searchParams.get("email")?.trim() || "",
    role: url.searchParams.get("role")?.trim() || "",
  };

  if (authConfig.tokenUrl && authConfig.clientId) {
    try {
      const tokenResponse = await exchangeCodeForToken(authConfig, code);
      const userInfo = tokenResponse.access_token
        ? await fetchUserInfo(authConfig, tokenResponse.access_token)
        : null;
      const idTokenClaims = tokenResponse.id_token
        ? await verifyIDToken(authConfig, tokenResponse.id_token)
        : null;
      profile = {
        sub: userInfo?.sub || idTokenClaims?.sub || profile.sub,
        name: userInfo?.name || idTokenClaims?.name || profile.name,
        email: userInfo?.email || idTokenClaims?.email || profile.email,
        role:
          userInfo?.role ||
          idTokenClaims?.role ||
          userInfo?.preferred_username ||
          idTokenClaims?.preferred_username ||
          profile.role,
      };
    } catch (error) {
      const message = error instanceof Error ? error.message : "oidc_token_exchange_failed";
      return NextResponse.redirect(
        new URL(`/login?error=${encodeURIComponent(message)}`, request.url),
        303,
      );
    }
  }

  const response = NextResponse.redirect(new URL("/", request.url), 303);
  response.cookies.set(
    AUTH_COOKIE_NAME,
    serializeSession({
      userId: profile.sub || `oidc_${code}`,
      name: profile.name || "OIDC Operator",
      email: profile.email || "oidc.user@creditflow.local",
      role: profile.role || "analyst",
      authMode: "oidc",
    }),
    {
      httpOnly: true,
      sameSite: "lax",
      path: "/",
      maxAge: 60 * 60 * 8,
    },
  );
  response.cookies.set(OIDC_STATE_COOKIE, "", {
    httpOnly: true,
    sameSite: "lax",
    path: "/",
    maxAge: 0,
  });

  return response;
}
