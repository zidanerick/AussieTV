'use strict';
const {test} = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const root = path.resolve(__dirname, '..');
const read = name => fs.readFileSync(path.join(root,name),'utf8');

test('pre-release version is consistent and clearly marked as beta', () => {
  assert.equal(read('RELEASE_VERSION').trim(), 'v0.1.0-beta.1');
  assert.match(read('main.go'), /const version = "0\.1\.0-beta\.1"/);
  assert.match(read('README.md'), /early public beta/i);
  assert.match(read('README.md'), /MPEG-TS.*not considered stable|MPEG-TS.*experimental/i);
});
test('installer has a real FFmpeg choice and separate install path', () => {
  const iss = read('installer/AussieTV.iss');
  assert.match(iss, /CreateInputDirPage/);
  assert.match(iss, /WizardIsTaskSelected\('ffmpeg'\)/);
  assert.match(iss, /fetch-windows\.ps1/);
  assert.match(iss, /DefaultDirName=\{localappdata\}/);
  assert.match(iss, /UninstallDisplayIcon=/);
  assert.match(read('tools/ffmpeg/fetch-windows.ps1'), /Get-FileHash -Algorithm SHA256/);
  assert.match(read('tools/ffmpeg/fetch-windows.ps1'), /ffmpeg-location\.txt/);
});
test('a single beta release has portable and installer packages', () => {
  const workflow = read('.github/workflows/beta-release.yml');
  assert.match(workflow, /AussieTV-Portable-v0\.1\.0-beta\.1-win-x64\.zip/);
  assert.match(workflow, /AussieTV-Setup-v0\.1\.0-beta\.1-win-x64\.exe/);
  assert.match(workflow, /--prerelease/);
  assert.match(workflow, /on:/);
  assert.match(read('RELEASE_NOTES.md'), /known problems/i);
});
