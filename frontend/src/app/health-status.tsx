import type { HealthResult } from "../lib/backend";

export function HealthStatus({ health }: { health: HealthResult }) {
  return (
    <section aria-labelledby="backend-heading">
      <h2 id="backend-heading">Backend health</h2>
      {health.available ? (
        <>
          <p role="status">Backend available</p>
          <pre>{JSON.stringify(health.data)}</pre>
        </>
      ) : (
        <p role="status">Backend unavailable. Reload the page to try again.</p>
      )}
    </section>
  );
}
