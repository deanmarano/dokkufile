import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  timeout: 300_000,
  retries: 0,
  workers: 1,
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
});
