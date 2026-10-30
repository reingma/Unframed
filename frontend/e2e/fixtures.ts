import { test as base, expect } from "@playwright/test";

import { BACKEND } from "./servers";

// Every page test asserts that the browser never contacts Go directly.
export const test = base.extend<{ backendRequests: string[] }>({
  backendRequests: [
    async ({ page }, use) => {
      const requests: string[] = [];
      const goPort = new URL(BACKEND).port;
      page.on("request", (req) => {
        if (new URL(req.url()).port === goPort) requests.push(req.url());
      });
      await use(requests);
      expect(requests).toEqual([]);
    },
    { auto: true },
  ],
});

export { expect };
