#!/usr/bin/env node
const child_process = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");

const platform = os.platform();
const arch = os.arch();
const binary = platform === "win32" ? "cosyra.exe" : "cosyra";
const candidate = path.join(__dirname, "..", "dist", `${platform}-${arch}`, binary);

if (!fs.existsSync(candidate)) {
  console.error(`cosyra binary not found for ${platform}-${arch}`);
  console.error("Build from source with: go build -o dist/cosyra ./cmd/cosyra");
  process.exit(1);
}

const result = child_process.spawnSync(candidate, process.argv.slice(2), {
  stdio: "inherit"
});

process.exit(result.status ?? 1);

