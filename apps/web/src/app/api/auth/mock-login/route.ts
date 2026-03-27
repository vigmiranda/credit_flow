import { NextResponse } from "next/server";

import { getAuthConfig } from "@/lib/auth/config";
import { AUTH_COOKIE_NAME, serializeSession } from "@/lib/auth/core";

export async function POST(request: Request) {
  const authConfig = getAuthConfig();
  if (authConfig.mode !== "mock") {
    return NextResponse.json(
      { message: "mock login indisponivel quando AUTH_MODE=oidc" },
      { status: 400 },
    );
  }

  const formData = await request.formData();
  const name = String(formData.get("name") ?? "").trim();
  const email = String(formData.get("email") ?? "").trim();
  const role = String(formData.get("role") ?? "analyst").trim();
  const userId = String(formData.get("user_id") ?? "").trim();

  if (!name || !email || !userId) {
    return NextResponse.redirect(new URL("/login", request.url), 303);
  }

  const response = NextResponse.redirect(new URL("/", request.url), 303);
  response.cookies.set(
    AUTH_COOKIE_NAME,
    serializeSession({
      userId,
      name,
      email,
      role,
      authMode: "mock",
    }),
    {
      httpOnly: true,
      sameSite: "lax",
      path: "/",
      maxAge: 60 * 60 * 8,
    },
  );
  return response;
}
