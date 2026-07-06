

const { existsSync, mkdirSync, createWriteStream, chmodSync, unlinkSync, copyFileSync, rmSync } = require("fs");
const path = require("path");
const https = require("https");
const { execSync } = require("child_process");

const VERSION = "0.4.6";
const REPO = "Encratahq/encrata-cli";

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  arm64: "arm64",
  x64: "amd64",
};

const platform = PLATFORM_MAP[process.platform];
const arch = ARCH_MAP[process.arch];

if (!platform || !arch) {
  console.error(`Unsupported platform: ${process.platform}-${process.arch}`);
  console.error("Install from source: go install github.com/Encratahq/cli@latest");
  process.exit(1);
}

const ext = process.platform === "win32" ? ".zip" : ".tar.gz";
// Name of the binary *inside* the GoReleaser archive.
const archiveBinName = process.platform === "win32" ? "encrata.exe" : "encrata";
// Name we store it as on disk — kept distinct from the committed `encrata`
// launcher shim so the download never clobbers the npm-linked bin entry.
const localBinName = process.platform === "win32" ? "encrata-bin.exe" : "encrata-bin";
const assetName = `encrata_${VERSION}_${platform}_${arch}${ext}`;
const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${assetName}`;

const destDir = path.join(__dirname, "bin");
const destPath = path.join(destDir, localBinName);

if (existsSync(destPath)) {
  process.exit(0);
}

if (!existsSync(destDir)) {
  mkdirSync(destDir, { recursive: true });
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const request = https.get(url, (res) => {
      if (res.statusCode === 302 || res.statusCode === 301) {
        return download(res.headers.location, dest).then(resolve).catch(reject);
      }
      if (res.statusCode !== 200) {
        return reject(new Error(`Download failed: HTTP ${res.statusCode}`));
      }
      const file = createWriteStream(dest);
      res.pipe(file);
      file.on("finish", () => file.close(resolve));
      file.on("error", reject);
    });
    request.on("error", reject);
  });
}

async function install() {
  const archivePath = path.join(destDir, assetName);
  // Extract into a temp subfolder so the archive's `encrata` binary never
  // overwrites the committed `encrata` launcher shim that npm links to PATH.
  const extractDir = path.join(destDir, ".extract");

  console.log(`Downloading encrata v${VERSION} for ${platform}/${arch}...`);
  await download(url, archivePath);

  mkdirSync(extractDir, { recursive: true });
  if (ext === ".tar.gz") {
    execSync(`tar -xzf "${archivePath}" -C "${extractDir}" ${archiveBinName}`, { stdio: "ignore" });
  } else {
    execSync(`powershell -command "Expand-Archive -Path '${archivePath}' -DestinationPath '${extractDir}' -Force"`, { stdio: "ignore" });
  }

  copyFileSync(path.join(extractDir, archiveBinName), destPath);
  chmodSync(destPath, 0o755);
  rmSync(extractDir, { recursive: true, force: true });
  unlinkSync(archivePath);
  console.log("encrata installed successfully.");
}

install().catch((err) => {
  console.error(`Failed to install encrata: ${err.message}`);
  console.error("Install from source: go install github.com/Encratahq/cli@latest");
  process.exit(1);
});
