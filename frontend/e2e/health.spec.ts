import { expect, test } from "@playwright/test";

test("renders the real Go health response fetched server-side", async ({
  page,
  request,
}) => {
  const response = await request.get("http://127.0.0.1:18080/healthz");
  expect(response.status()).toBe(200);
  expect(response.headers()["content-type"]).toBe("application/json");
  const health: unknown = await response.json();
  expect(health).toEqual({ status: "ok" });

  const backendRequests: string[] = [];
  page.on("request", (req) => {
    if (new URL(req.url()).port === "18080") backendRequests.push(req.url());
  });
  await page.goto("http://127.0.0.1:13000");
  await expect(
    page.getByRole("heading", { name: "Unframed", exact: true }),
  ).toBeVisible();
  await expect(page.getByRole("status")).toHaveText("Backend available");
  await expect(page.locator("pre")).toHaveText(JSON.stringify(health));
  expect(backendRequests).toEqual([]);
});

test("renders the fallback when the backend is unreachable", async ({
  page,
}) => {
  await page.goto("http://127.0.0.1:13001");
  await expect(page.getByRole("status")).toHaveText(
    "Backend unavailable. Reload the page to try again.",
  );
  await expect(page.locator("pre")).toHaveCount(0);
});

test("routes only the health endpoint and rejects unsupported methods", async ({
  request,
}) => {
  expect((await request.get("http://127.0.0.1:18080/missing")).status()).toBe(
    404,
  );
  const response = await request.post("http://127.0.0.1:18080/healthz");
  expect(response.status()).toBe(405);
  expect(response.headers()["allow"]).toContain("GET");
});
