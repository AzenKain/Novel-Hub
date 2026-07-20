const fs = require('fs');
const path = require('path');

function walkDir(dir, callback) {
  fs.readdirSync(dir).forEach(f => {
    let dirPath = path.join(dir, f);
    let isDirectory = fs.statSync(dirPath).isDirectory();
    isDirectory ? 
      walkDir(dirPath, callback) : callback(path.join(dir, f));
  });
}

const srcDir = path.join(__dirname, 'src');

walkDir(srcDir, function(filePath) {
  if (filePath.endsWith('.ts') || filePath.endsWith('.tsx')) {
    let content = fs.readFileSync(filePath, 'utf-8');
    
    content = content.replace(/from\s+['"]\.\.\/\.\.\/(.*?)['"]/g, 'from "@/$1"');
    content = content.replace(/from\s+['"]\.\.\/(.*?)['"]/g, 'from "@/$1"');
    
    if (path.dirname(filePath) === srcDir) {
        content = content.replace(/from\s+['"]\.\/(.*?)['"]/g, 'from "@/$1"');
    }
    
    fs.writeFileSync(filePath, content, 'utf-8');
  }
});
console.log("Refactoring complete");
