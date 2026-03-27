import { randomUUID } from "crypto";

import { NextResponse } from "next/server";

import { getAuthConfig } from "@/lib/auth/config";

const OIDC_STATE_COOKIE = "credit_flow_oidc_state";

export async function GET(request: Request) {
  const authConfig = getAuthConfig();
  if (authConfig.mode !== "oidc" || !authConfig.authorizeUrl || !authConfig.clientId) {
    return NextResponse.redirect(new URL("/login?error=oidc_not_configured", request.url), 303);
  }

  const state = randomUUID();
  const authorizeURL = new URL(authConfig.authorizeUrl);
  authorizeURL.searchParams.set("response_type", "code");
  authorizeURL.searchParams.set("client_id", authConfig.clientId);
  authorizeURL.searchParams.set("redirect_uri", authConfig.redirectUri);
  authorizeURL.searchParams.set("scope", "openid profile email");
  authorizeURL.searchParams.set("state", state);

  const response = NextResponse.redirect(authorizeURL, 303);
  response.cookies.set(OIDC_STATE_COOKIE, state, {
    httpOnly: true,
    sameSite: "lax",
    path: "/",
    maxAge: 60 * 10,
  });
  return response;
}
