import { test, expect } from "@playwright/test";
import { dokkufile, dokku, appExists, cleanupApp } from "./helpers.js";

test.describe("dokkufile plan", () => {
  // TODO: These tests require a running dokku server.
  // Skipped until E2E environment is available.

  test.skip(true, "requires dokku server");

  test("plan detects drift when app is missing", () => {
    // Create a Dokkufile that declares an app not on the server
    const output = dokkufile("plan -f fixtures/new-app.yml", {
      ignoreError: true,
    });
    expect(output).toContain("+ app");
  });

  test("plan shows no changes when state matches", () => {
    const output = dokkufile("plan -f fixtures/matching.yml", {
      ignoreError: true,
    });
    expect(output).toContain("No changes needed");
  });
});
