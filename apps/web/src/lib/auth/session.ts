import { cookies } from "next/headers";

import { AUTH_COOKIE_NAME, parseSession, type AuthSession } from "@/lib/auth/core";

export async function getServerSession(): Promise<AuthSession | null> {
  const cookieStore = await cookies();
  return parseSession(cookieStore.get(AUTH_COOKIE_NAME)?.value);
}
