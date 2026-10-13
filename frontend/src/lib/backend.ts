import "server-only";

import { z } from "zod";

export const healthSchema = z.object({ status: z.literal("ok") });
export type HealthResponse = z.infer<typeof healthSchema>;

export type HealthResult =
  { available: true; data: HealthResponse } | { available: false };

export async function getBackendHealth(): Promise<HealthResult> {
  try {
    const baseUrl = process.env.BACKEND_URL || "http://127.0.0.1:8080";
    const response = await fetch(new URL("/healthcheck", baseUrl), {
      cache: "no-store",
      signal: AbortSignal.timeout(2_000),
    });
    if (!response.ok) return { available: false };

    const data = healthSchema.safeParse(await response.json());
    return data.success
      ? { available: true, data: data.data }
      : { available: false };
  } catch {
    return { available: false };
  }
}
