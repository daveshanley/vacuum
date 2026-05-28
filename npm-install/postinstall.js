import { createWriteStream } from "fs";
import * as fs from "fs/promises";
import https from "https";
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

function createRequestOptions() {
    const proxyUrl = getProxyUrl();
    if (!proxyUrl) {
        return {};
    }
    console.log("Using proxy:", proxyUrl);
    return { agent: new HttpsProxyAgent(proxyUrl) };
}

function requestUrl(url, options, redirectsRemaining = 5) {
    return new Promise((resolve, reject) => {
        const request = https.get(url, options, (response) => {
            const { statusCode, headers } = response;
            const location = headers.location;

            if (statusCode >= 300 && statusCode < 400 && location) {
                response.resume();
                if (redirectsRemaining <= 0) {
                    reject(new Error("Too many redirects while fetching the binary"));
                    return;
                }
                const nextUrl = new URL(location, url).toString();
                requestUrl(nextUrl, options, redirectsRemaining - 1).then(resolve, reject);
                return;
            }

            resolve(response);
        });
        request.on("error", reject);
    });
}

async function downloadFile(url, destination) {
    const response = await requestUrl(url, createRequestOptions());
    if (response.statusCode < 200 || response.statusCode > 299) {
        response.resume();
        throw new Error("Failed fetching the binary: " + response.statusMessage);
    }
    await pipeline(response, createWriteStream(destination));
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

    const tarFile = "downloaded.tar.gz";

    await fs.mkdir(binPath, { recursive: true });
    console.log("fetching from URL", url);
    await downloadFile(url, tarFile);
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
