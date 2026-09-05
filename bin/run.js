#!/usr/bin/env node

/**
 * birdnet-go-mcp npx runner
 * Detects platform/arch, downloads pre-compiled native Go binary on demand,
 * caches it locally, and executes it with inherited stdio for pure JSON-RPC / CLI.
 */

const os = require('os');
const fs = require('fs');
const path = require('path');
const https = require('https');
const { spawn } = require('child_process');

const pkg = require('../package.json');
const VERSION = pkg.version;
const REPO = 'zax0rz/birdnet-go-mcp';

function getBinaryName() {
  const platform = os.platform();
  const arch = os.arch();

  let osName = '';
  let archName = '';

  if (platform === 'darwin') {
    osName = 'darwin';
  } else if (platform === 'linux') {
    osName = 'linux';
  } else if (platform === 'win32') {
    osName = 'windows';
  } else {
    throw new Error(`Unsupported operating system: ${platform}`);
  }

  if (arch === 'x64') {
    archName = 'amd64';
  } else if (arch === 'arm64') {
    archName = 'arm64';
  } else if (arch === 'arm') {
    archName = 'armv7';
  } else {
    throw new Error(`Unsupported architecture: ${arch}`);
  }

  const ext = platform === 'win32' ? '.exe' : '';
  return `birdnet-mcp-${osName}-${archName}${ext}`;
}

function getCacheDir() {
  const home = os.homedir();
  if (os.platform() === 'win32') {
    return path.join(process.env.LOCALAPPDATA || path.join(home, 'AppData', 'Local'), 'birdnet-go-mcp');
  }
  return path.join(home, '.cache', 'birdnet-go-mcp');
}

function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    https.get(url, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        // Follow redirect
        return downloadFile(res.headers.location, dest).then(resolve).catch(reject);
      }

      if (res.statusCode !== 200) {
        return reject(new Error(`Download failed with HTTP ${res.statusCode} from ${url}`));
      }

      const file = fs.createWriteStream(dest, { mode: 0o755 });
      res.pipe(file);
      file.on('finish', () => file.close(resolve));
      file.on('error', (err) => {
        fs.unlink(dest, () => reject(err));
      });
    }).on('error', reject);
  });
}

async function resolveBinary() {
  const binName = getBinaryName();

  // 1. Check if binary exists in local repo dist/ (development mode)
  const localDistBin = path.join(__dirname, '..', 'dist', binName);
  if (fs.existsSync(localDistBin)) {
    return localDistBin;
  }

  // 2. Check if cached binary exists
  const cacheDir = path.join(getCacheDir(), `v${VERSION}`);
  const cachedBin = path.join(cacheDir, binName);
  if (fs.existsSync(cachedBin)) {
    return cachedBin;
  }

  // 3. Download from GitHub Releases
  fs.mkdirSync(cacheDir, { recursive: true });
  const downloadUrl = `https://github.com/${REPO}/releases/download/v${VERSION}/${binName}`;
  const tmpDest = path.join(cacheDir, `${binName}.tmp`);

  process.stderr.write(`[birdnet-go-mcp] Downloading native binary for ${os.platform()}-${os.arch()} (v${VERSION})...\n`);

  try {
    await downloadFile(downloadUrl, tmpDest);
    fs.chmodSync(tmpDest, 0o755);

    // On macOS Apple Silicon / Intel, ad-hoc codesign the binary to satisfy AMFI
    if (os.platform() === 'darwin') {
      try {
        const { execSync } = require('child_process');
        execSync(`codesign -s - --force "${tmpDest}"`, { stdio: 'ignore' });
      } catch (_) {}
    }

    fs.renameSync(tmpDest, cachedBin);
    return cachedBin;
  } catch (err) {
    if (fs.existsSync(tmpDest)) {
      fs.unlinkSync(tmpDest);
    }
    throw new Error(`Failed to download binary: ${err.message}\nMake sure your platform (${os.platform()}-${os.arch()}) is supported in release v${VERSION}.`);
  }
}

async function main() {
  try {
    const binPath = await resolveBinary();
    const args = process.argv.slice(2);

    const child = spawn(binPath, args, {
      stdio: 'inherit',
      env: process.env,
    });

    child.on('error', (err) => {
      console.error(`[birdnet-go-mcp] Failed to run binary: ${err.message}`);
      process.exit(1);
    });

    child.on('close', (code) => {
      process.exit(code || 0);
    });

    // Forward signals
    process.on('SIGINT', () => child.kill('SIGINT'));
    process.on('SIGTERM', () => child.kill('SIGTERM'));
  } catch (err) {
    console.error(`[birdnet-go-mcp] Error: ${err.message}`);
    process.exit(1);
  }
}

main();
