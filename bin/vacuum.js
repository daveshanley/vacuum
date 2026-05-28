#!/usr/bin/env node
import { execFileSync } from "child_process";
import path from "path";
import { exit } from "process";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const binaryName = process.platform === "win32" ? "vacuum.exe" : "vacuum";

try {
    const env = {
        ...process.env,
        VACUUM_MANAGED_BY_NPM: "1",
        VACUUM_MANAGED_PACKAGE_ROOT: path.resolve(`${__dirname}/..`),
    };
    execFileSync(path.resolve(__dirname, binaryName), process.argv.slice(2), {
        stdio: "inherit",
        env,
    });
} catch (e) {
    exit(typeof e.status === "number" ? e.status : 1)
}
