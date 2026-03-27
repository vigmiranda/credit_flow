import { NextResponse } from "next/server";

export async function GET(request: Request) {
  return NextResponse.json(
    {
      status: "not_implemented",
      message: "callback OIDC ainda nao implementado neste ciclo",
      next: new URL("/login", request.url).toString(),
    },
    { status: 501 },
  );
}
