import { createWriteStream } from "fs";
import * as fs from "fs/promises";
import fetch from "node-fetch";
import { HttpsProxyAgent } from "https-proxy-agent";
import { pipeline } from "stream/promises";
import * as tar from "tar";
import { execSync } from "child_process";

import { ARCH_MAPPING, CONFIG, PLATFORM_MAPPING } from "./config.js";

// Get proxy URL from environment variables (standard convention)
function getProxyUrl() {
    // Check common proxy environment variables (case-insensitive on some systems)
    return process.env.HTTPS_PROXY ||
           process.env.https_proxy ||
           process.env.HTTP_PROXY ||
           process.env.http_proxy ||
           null;
}

async function install() {
    if (process.platform === "android") {
        console.log("Installing, may take a moment...");
        const cmd =
            "pkg upgrade && pkg install golang git -y && git clone https://github.com/daveshanley/vacuum.git && cd cli/ && go build -o $PREFIX/bin/vacuum";
        execSync(cmd, { encoding: "utf-8" });
        console.log("Installation successful!");
        return;
    }
    const packageJson = await fs.readFile("package.json").then(JSON.parse);
    let version = packageJson.version;

    if (typeof version !== "string") {
        throw new Error("Missing version in package.json");
    }

    if (version[0] === "v") version = version.slice(1);

    let { name: binName, path: binPath, url } = CONFIG;

    url = url.replace(/{{arch}}/g, ARCH_MAPPING[process.arch]);
    url = url.replace(/{{platform}}/g, PLATFORM_MAPPING[process.platform]);
    url = url.replace(/{{version}}/g, version);
    url = url.replace(/{{bin_name}}/g, binName);

    // Configure fetch options with proxy support
    const fetchOptions = {};
    const proxyUrl = getProxyUrl();
    if (proxyUrl) {
        console.log('Using proxy:', proxyUrl);
        fetchOptions.agent = new HttpsProxyAgent(proxyUrl);
    }

    console.log('fetching from URL', url)
    const response = await fetch(url, fetchOptions);
    if (!response.ok) {
        throw new Error("Failed fetching the binary: " + response.statusText);
    }

    const tarFile = "downloaded.tar.gz";

    await fs.mkdir(binPath, { recursive: true });
    await pipeline(response.body, createWriteStream(tarFile));
    await tar.x({ file: tarFile, cwd: binPath });
    await fs.rm(tarFile);
}

install()
    .then(async () => {
        process.exit(0);
    })
    .catch(async (err) => {
        console.error(err);
        process.exit(1);
    });