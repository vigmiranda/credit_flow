import HomePage from "@/components/home-page";
import { getServerSession } from "@/lib/auth/session";

export default async function Page() {
  const session = await getServerSession();

  if (!session) {
    return null;
  }

  return <HomePage session={session} />;
}
