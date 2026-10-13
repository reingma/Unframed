// @vitest-environment jsdom

import "@testing-library/jest-dom/vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { HealthStatus } from "./health-status";

afterEach(cleanup);

describe("HealthStatus", () => {
  it("displays the validated response", () => {
    render(
      <HealthStatus health={{ available: true, data: { status: "ok" } }} />,
    );
    expect(screen.getByRole("status")).toHaveTextContent("Backend available");
    expect(screen.getByText('{"status":"ok"}')).toBeVisible();
  });

  it("displays the fallback without healthy data", () => {
    render(<HealthStatus health={{ available: false }} />);
    expect(screen.getByRole("status")).toHaveTextContent("Backend unavailable");
    expect(screen.queryByText('{"status":"ok"}')).not.toBeInTheDocument();
  });
});
