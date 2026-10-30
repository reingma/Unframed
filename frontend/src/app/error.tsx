"use client";

// Error presents a backend that is down or answered outside the contract.
// The server has already logged the cause with the request's correlation ID.
export default function Error({
  error,
}: {
  error: Error & { digest?: string };
}) {
  return (
    <main className="status-page">
      <h1>Content unavailable</h1>
      <p>The site could not load this page. Reload to try again.</p>
      {error.digest && <p>Reference: {error.digest}</p>}
    </main>
  );
}
