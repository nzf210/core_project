const fs = require('fs');
const path = require('path');

const srcDir = path.join(__dirname, 'frontend/campaign-web/src/components');
const files = fs.readdirSync(srcDir).filter(f => f.endsWith('.vue'));

files.forEach(file => {
  const filePath = path.join(srcDir, file);
  let content = fs.readFileSync(filePath, 'utf8');

  // Insert import
  if (content.includes('http://localhost:8301') && !content.includes('import { apiClient }')) {
    content = content.replace(/<script setup lang="ts">([\s\S]*?)from 'vue'/m, `<script setup lang="ts">\nimport { apiClient } from '../api'\n$1from 'vue'`);
    // Some files might just have import { ref } from 'vue'
    if (!content.includes("import { apiClient }")) {
      content = content.replace(/<script setup lang="ts">/, `<script setup lang="ts">\nimport { apiClient } from '../api'`);
    }
  }

  // Replace fetch
  content = content.replace(/fetch\('http:\/\/localhost:8301\/([^']+)'/g, "apiClient('/$1'");
  content = content.replace(/fetch\(`http:\/\/localhost:8301\/([^`]+)`/g, "apiClient(`/$1`");

  fs.writeFileSync(filePath, content, 'utf8');
  console.log(`Refactored ${file}`);
});
