import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { getAuthConfig } from "@/lib/auth/config";
import { AUTH_COOKIE_NAME, serializeSession } from "@/lib/auth/core";

const OIDC_STATE_COOKIE = "credit_flow_oidc_state";

export async function GET(request: Request) {
  const authConfig = getAuthConfig()
  const url = new URL(request.url)
  const code = url.searchParams.get("code")?.trim()
  const state = url.searchParams.get("state")?.trim()
  const authError = url.searchParams.get("error")?.trim()

  if (authConfig.mode !== "oidc") {
    return NextResponse.redirect(new URL("/login", request.url), 303)
  }

  if (authError) {
    return NextResponse.redirect(new URL(`/login?error=${encodeURIComponent(authError)}`, request.url), 303)
  }

  const cookieStore = await cookies()
  const expectedState = cookieStore.get(OIDC_STATE_COOKIE)?.value
  if (!code || !state || !expectedState || state !== expectedState) {
    return NextResponse.redirect(new URL("/login?error=invalid_oidc_callback", request.url), 303)
  }

  const response = NextResponse.redirect(new URL("/", request.url), 303)
  response.cookies.set(
    AUTH_COOKIE_NAME,
    serializeSession({
      userId: url.searchParams.get("sub")?.trim() || `oidc_${code}`,
      name: url.searchParams.get("name")?.trim() || "OIDC Operator",
      email: url.searchParams.get("email")?.trim() || "oidc.user@creditflow.local",
      role: url.searchParams.get("role")?.trim() || "analyst",
      authMode: "oidc",
    }),
    {
      httpOnly: true,
      sameSite: "lax",
      path: "/",
      maxAge: 60 * 60 * 8,
    },
  )
  response.cookies.set(OIDC_STATE_COOKIE, "", {
    httpOnly: true,
    sameSite: "lax",
    path: "/",
    maxAge: 0,
  })

  return response
}
