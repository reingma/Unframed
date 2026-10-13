import { getBackendHealth } from "../lib/backend";
import { HealthStatus } from "./health-status";

export const dynamic = "force-dynamic";

export default async function Home() {
  const health = await getBackendHealth();

  return (
    <main>
      <p className="eyebrow">Engineering, without the frame.</p>
      <h1>Unframed</h1>
      <p>A Go and Next.js skeleton. The blog starts here.</p>
      <HealthStatus health={health} />
    </main>
  );
}
