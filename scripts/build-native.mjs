import { execSync } from "child_process";
import fs from "fs";
import path from "path";

const rootDir = process.cwd();
const distDir = path.join(rootDir, "dist-native");

console.log("=== [1/3] Building Web Frontend ===");
try {
  const bunCmd = process.platform === "win32" ? "bun.exe" : "bun";
  execSync(`${bunCmd} install --frozen-lockfile`, {
    cwd: path.join(rootDir, "web"),
    stdio: "inherit",
  });
  execSync(`${bunCmd} run build`, {
    cwd: path.join(rootDir, "web"),
    stdio: "inherit",
  });
} catch (error) {
  console.error("Frontend build failed:", error.message);
  process.exit(1);
}

if (!fs.existsSync(distDir)) {
  fs.mkdirSync(distDir, { recursive: true });
}

const targets = {
  windows: [
    { GOOS: "windows", GOARCH: "amd64", filename: "novelhub-windows-amd64.exe" },
    { GOOS: "windows", GOARCH: "arm64", filename: "novelhub-windows-arm64.exe" },
  ],
  linux: [
    { GOOS: "linux", GOARCH: "amd64", filename: "novelhub-linux-amd64" },
    { GOOS: "linux", GOARCH: "arm64", filename: "novelhub-linux-arm64" },
  ],
  mac: [
    { GOOS: "darwin", GOARCH: "amd64", filename: "novelhub-darwin-amd64" },
    { GOOS: "darwin", GOARCH: "arm64", filename: "novelhub-darwin-arm64" },
  ],
};

const argTarget = process.argv[2] || "all";
let releaseVersion = (process.env.RELEASE_VERSION || "").trim();

if (!releaseVersion && fs.existsSync(path.join(rootDir, "release/release.json"))) {
  try {
    const meta = JSON.parse(fs.readFileSync(path.join(rootDir, "release/release.json"), "utf8"));
    if (meta.version) {
      releaseVersion = String(meta.version).trim().replace(/^v/i, "");
    }
  } catch (e) {}
}

console.log(`\n=== [2/3] Cross-compiling Go Binaries (Version: ${releaseVersion || "dev"}) ===`);

function buildBinary(config) {
  const outputPath = path.join(distDir, config.filename);
  console.log(`Building OS=${config.GOOS} Arch=${config.GOARCH} -> ${config.filename}...`);

  try {
    const versionFlag = releaseVersion ? ` -X main.Version=${releaseVersion}` : "";
    execSync(`go build -trimpath -ldflags="-s -w${versionFlag}" -o "${outputPath}" ./cmd/api`, {
      cwd: rootDir,
      env: {
        ...process.env,
        CGO_ENABLED: "0",
        GOOS: config.GOOS,
        GOARCH: config.GOARCH,
      },
      stdio: "inherit",
    });
    console.log(`✓ Successfully built ${config.filename}`);
  } catch (error) {
    console.error(`✗ Failed to build for OS=${config.GOOS} Arch=${config.GOARCH}:`, error.message);
    process.exit(1);
  }
}

if (argTarget === "all") {
  for (const platform of Object.keys(targets)) {
    for (const config of targets[platform]) {
      buildBinary(config);
    }
  }
} else if (targets[argTarget]) {
  for (const config of targets[argTarget]) {
    buildBinary(config);
  }
} else {
  console.error(`Unknown build target: "${argTarget}". Use: windows, linux, mac, or all.`);
  process.exit(1);
}

console.log("\n=== [3/3] Build Completed Successfully! ===");
console.log(`Artifacts location: ${distDir}`);
