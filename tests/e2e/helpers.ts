import { execSync } from "node:child_process";
import { resolve } from "node:path";

export interface DokkufileOptions {
  timeout?: number;
  ignoreError?: boolean;
}

const DOKKUFILE_BIN = resolve(__dirname, "../../dokkufile");

/**
 * Run the dokkufile CLI with the given arguments.
 */
export function dokkufile(
  args: string,
  opts: DokkufileOptions = {}
): string {
  const timeout = opts.timeout ?? 30_000;
  try {
    const output = execSync(`${DOKKUFILE_BIN} ${args}`, {
      timeout,
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "pipe"],
    });
    return output.trim();
  } catch (err: unknown) {
    if (opts.ignoreError) {
      const e = err as { stdout?: string; stderr?: string };
      return (e.stdout ?? "") + (e.stderr ?? "");
    }
    throw err;
  }
}

/**
 * Run a dokku command on the server.
 */
export function dokku(
  cmd: string,
  opts: DokkufileOptions = {}
): string {
  const timeout = opts.timeout ?? 60_000;
  try {
    const output = execSync(`sudo bash -c 'dokku ${cmd}'`, {
      timeout,
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "pipe"],
    });
    return output.trim();
  } catch (err: unknown) {
    if (opts.ignoreError) {
      const e = err as { stdout?: string; stderr?: string };
      return (e.stdout ?? "") + (e.stderr ?? "");
    }
    throw err;
  }
}

/**
 * Check if an app exists on the dokku server.
 */
export function appExists(appName: string): boolean {
  try {
    dokku(`apps:exists ${appName}`);
    return true;
  } catch {
    return false;
  }
}

/**
 * Destroy a dokku app (best-effort cleanup).
 */
export function cleanupApp(appName: string): void {
  dokku(`apps:destroy ${appName} --force`, { ignoreError: true });
}

/**
 * Wait for an app to report as running.
 */
export async function waitForHealthy(
  appName: string,
  timeout = 60_000
): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    try {
      const output = dokku(`ps:report ${appName} --running`);
      if (output.trim() === "true") return true;
    } catch {
      // not ready yet
    }
    await new Promise((r) => setTimeout(r, 5_000));
  }
  return false;
}
