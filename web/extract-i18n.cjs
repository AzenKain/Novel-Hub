const fs = require('fs');
const path = require('path');

const srcDir = path.join(__dirname, 'src');
const localesDir = path.join(__dirname, 'public', 'locales');

function findFiles(dir, fileList = []) {
  const files = fs.readdirSync(dir);
  for (const file of files) {
    const filePath = path.join(dir, file);
    if (fs.statSync(filePath).isDirectory()) {
      findFiles(filePath, fileList);
    } else if (filePath.endsWith('.ts') || filePath.endsWith('.tsx')) {
      fileList.push(filePath);
    }
  }
  return fileList;
}

const allFiles = findFiles(srcDir);
const keys = {};

// Match t('namespace.key', 'fallback')
const regex = /t\(['"]([^'"]+)['"]\s*,\s*['"]([^'"]+)['"]\)/g;

allFiles.forEach(file => {
  const content = fs.readFileSync(file, 'utf-8');
  let match;
  while ((match = regex.exec(content)) !== null) {
    const fullKey = match[1];
    const fallback = match[2];
    
    const parts = fullKey.split('.');
    if (parts.length === 2) {
      const ns = parts[0];
      const key = parts[1];
      if (!keys[ns]) keys[ns] = {};
      keys[ns][key] = fallback;
    }
  }
});

const enFile = path.join(localesDir, 'en.json');
let enData = {};
if (fs.existsSync(enFile)) {
  enData = JSON.parse(fs.readFileSync(enFile, 'utf-8'));
}

// Merge keeping existing keys if they exist (except we want to ensure we have all fallbacks if missing)
for (const ns in keys) {
  if (!enData[ns]) enData[ns] = {};
  for (const key in keys[ns]) {
    if (!enData[ns][key]) {
      enData[ns][key] = keys[ns][key];
    }
  }
}

fs.writeFileSync(enFile, JSON.stringify(enData, null, 2));
console.log('Extracted and merged keys into en.json');
