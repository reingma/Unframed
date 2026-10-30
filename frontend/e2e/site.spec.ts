import { expect, test } from "./fixtures";
import { PREVIEW_TOKEN, SITE, SITE_WITHOUT_BACKEND } from "./servers";

const navLinks = (page: import("@playwright/test").Page) =>
  page.getByRole("navigation", { name: "Site" }).getByRole("link");

test("renders a published article with its metadata and inherited theme", async ({
  page,
}) => {
  const response = await page.goto(`${SITE}/guides/alpha`);
  expect(response?.status()).toBe(200);
  await expect(page).toHaveTitle("Alpha | Fixture Site");
  await expect(page.locator('meta[name="description"]')).toHaveAttribute(
    "content",
    "Alpha description",
  );
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Alpha guide",
  );
  await expect(page.getByText("Alpha body text.")).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Alpha", exact: true }),
  ).toHaveAttribute("aria-current", "page");

  const site = page.locator(".site");
  await expect(site).toHaveCSS("--accent", "#aa3300");
  await expect(site).toHaveCSS("--background", "#fffdf8");
  await expect(site).toHaveCSS("background-color", "rgb(255, 253, 248)");
});

test("lists the root without index.md and navigates the tree in order", async ({
  page,
}) => {
  const response = await page.goto(SITE);
  expect(response?.status()).toBe(200);
  await expect(page).toHaveTitle("Fixture Site");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Fixture Site",
  );
  await expect(page.locator(".listing li > a")).toHaveText(["Guides", "Notes"]);
  await expect(navLinks(page)).toHaveText([
    "Guides",
    "Zeta",
    "Alpha",
    "Notes",
    "Alpha Note",
    "Beta Note",
  ]);
  await expect(page.locator(".site")).toHaveCSS("--accent", "#336699");

  await page
    .getByRole("navigation")
    .getByRole("link", { name: "Guides" })
    .click();
  await expect(page).toHaveURL(`${SITE}/guides`);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Guides home",
  );

  await page
    .getByRole("navigation")
    .getByRole("link", { name: "Notes" })
    .click();
  await expect(page).toHaveURL(`${SITE}/notes`);
  await expect(page.locator(".listing li > a")).toHaveText([
    "Alpha Note",
    "Beta Note",
  ]);
  await expect(page.getByText("Published note")).toBeVisible();

  await page
    .locator(".listing")
    .getByRole("link", { name: "Alpha Note" })
    .click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Alpha note published",
  );
});

for (const [name, path] of [
  ["an invalid item", "/notes/bad-config"],
  ["a draft", "/guides/setup"],
  ["a draft's alias", "/guides/setup-old"],
  ["a path below an article alias", "/guides/first/more"],
  ["a looping alias", "/notes/alpha-note/x"],
  ["an unknown path", "/notes/missing"],
  ["a path that is not a slug", "/Notes"],
  ["the preview route without a token", "/preview"],
]) {
  test(`answers ${name} with a 404`, async ({ page }) => {
    const response = await page.goto(`${SITE}${path}`);
    expect(response?.status()).toBe(404);
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(
      "Page not found",
    );
    await expect(page.locator('meta[name="robots"]')).toHaveAttribute(
      "content",
      "noindex",
    );
  });
}

for (const [name, from, to] of [
  ["a renamed article", "/guides/first", "/guides/alpha"],
  ["a renamed directory", "/journal", "/notes"],
  [
    "a path below a renamed directory",
    "/journal/beta-note",
    "/notes/beta-note",
  ],
]) {
  test(`redirects ${name} permanently`, async ({ page, request }) => {
    const response = await request.get(`${SITE}${from}`, { maxRedirects: 0 });
    expect(response.status()).toBe(308);
    expect(response.headers()["location"]).toBe(to);

    const landed = await page.goto(`${SITE}${from}`);
    expect(landed?.status()).toBe(200);
    await expect(page).toHaveURL(`${SITE}${to}`);
  });
}

test("reports a redirect loop in diagnostics", async ({ page }) => {
  await page.goto(`${SITE}/preview/${PREVIEW_TOKEN}/diagnostics`);
  for (const path of ["Notes/Alpha-Note.md", "Notes/Beta-Note.md"]) {
    await expect(
      page.locator(".diagnostics li").filter({ hasText: path }),
    ).toContainText("redirect loop");
  }
  const response = await page.goto(`${SITE}/notes/alpha-note`);
  expect(response?.status()).toBe(200);
});

test("shows the error page when the backend is unreachable", async ({
  page,
}) => {
  const response = await page.goto(SITE_WITHOUT_BACKEND);
  expect(response?.status()).toBe(500);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Content unavailable",
  );
  await expect(page.locator(".site")).toHaveCount(0);
});
