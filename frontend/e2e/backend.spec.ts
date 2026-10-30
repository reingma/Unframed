import { expect, test } from "@playwright/test";

import { BACKEND } from "./servers";

test("Go answers only its routes and rejects unsupported methods", async ({
  request,
}) => {
  const health = await request.get(`${BACKEND}/healthcheck`);
  expect(health.status()).toBe(200);
  expect(await health.json()).toEqual({ status: "ok" });
  expect((await request.get(`${BACKEND}/missing`)).status()).toBe(404);
  const response = await request.post(`${BACKEND}/healthcheck`);
  expect(response.status()).toBe(405);
  expect(response.headers()["allow"]).toContain("GET");
});
