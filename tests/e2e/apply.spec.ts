import { test, expect } from "@playwright/test";
import { dokkufile, dokku, appExists, cleanupApp } from "./helpers.js";

const APP_NAME = "test-dokkufile-apply";

test.describe("dokkufile apply", () => {
  // TODO: These tests require a running dokku server.
  // Skipped until E2E environment is available.

  test.skip(true, "requires dokku server");

  test.afterAll(() => {
    cleanupApp(APP_NAME);
  });

  test("apply creates a new app", () => {
    const output = dokkufile("apply -f fixtures/new-app.yml");
    expect(output).toContain("create app");
    expect(appExists(APP_NAME)).toBe(true);
  });

  test("apply is idempotent", () => {
    const output = dokkufile("apply -f fixtures/new-app.yml");
    expect(output).toContain("No changes needed");
  });
});
