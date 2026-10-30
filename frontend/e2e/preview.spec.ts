import { expect, test } from "./fixtures";
import { PREVIEW_TOKEN, SITE } from "./servers";

const preview = `${SITE}/preview/${PREVIEW_TOKEN}`;

test("renders a draft only through the preview path", async ({ page }) => {
  const response = await page.goto(`${preview}/articles/guides/setup`);
  expect(response?.status()).toBe(200);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Setup draft",
  );
  await expect(page.getByText("Draft", { exact: true })).toBeVisible();
  await expect(page.getByText("Preview", { exact: true })).toBeVisible();
  await expect(page.locator('meta[name="robots"]')).toHaveAttribute(
    "content",
    "noindex, nofollow",
  );
  await expect(page.locator('meta[name="referrer"]')).toHaveAttribute(
    "content",
    "no-referrer",
  );

  expect((await page.goto(`${SITE}/guides/setup`))?.status()).toBe(404);
});

test("prefers the draft of a published article and keeps navigation in preview", async ({
  page,
}) => {
  await page.goto(`${preview}/articles/notes/alpha-note`);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Alpha note draft",
  );
  const nav = page.getByRole("navigation", { name: "Site" });
  for (const href of await nav
    .getByRole("link")
    .evaluateAll((as) => as.map((a) => a.getAttribute("href")))) {
    expect(href).toMatch(new RegExp(`^/preview/${PREVIEW_TOKEN}/articles/`));
  }

  await nav.getByRole("link", { name: "Notes" }).click();
  await expect(page).toHaveURL(`${preview}/articles/notes`);
  await expect(page.locator(".listing li > a")).toHaveText([
    "Alpha Note",
    "Beta Note",
  ]);
  await page.getByRole("link", { name: "Fixture Site" }).click();
  await expect(page).toHaveURL(`${preview}/articles`);
});

test("shows Go's diagnostics for an invalid item", async ({ page }) => {
  await page.goto(`${preview}/articles/notes/bad-config`);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Nothing to preview at /notes/bad-config",
  );
  await expect(page.locator(".diagnostics li")).toContainText([
    "Notes/Bad-Config.md",
  ]);
});

test("lists every build diagnostic", async ({ page }) => {
  const response = await page.goto(`${preview}/diagnostics`);
  expect(response?.status()).toBe(200);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Diagnostics",
  );
  await expect(page.getByText(/3 validation errors/)).toBeVisible();
  await expect(page.locator(".diagnostics li")).toHaveCount(3);
});

for (const path of ["/articles/guides/setup", "/diagnostics"]) {
  test(`hides ${path} behind a wrong token`, async ({ page }) => {
    for (const token of ["x".repeat(40), "short"]) {
      const response = await page.goto(`${SITE}/preview/${token}${path}`);
      expect(response?.status()).toBe(404);
      await expect(page.getByRole("heading", { level: 1 })).toHaveText(
        "Page not found",
      );
    }
  });
}
