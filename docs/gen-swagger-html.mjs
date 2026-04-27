import { readFileSync, writeFileSync, mkdirSync } from 'fs';
import { dirname } from 'path';

const SWAGGER_JSON = '/mnt/HC_Volume_103451728/eegabrechnung/api/docs/swagger.json';
const OUT_HTML = '/mnt/HC_Volume_103451728/eegabrechnung/docs/B-api-swagger.html';

// Read swagger.json
const specJson = readFileSync(SWAGGER_JSON, 'utf-8');

// Base64-encode the JSON
const specBase64 = Buffer.from(specJson).toString('base64');

// HTML template
const html = `<!DOCTYPE html>
<html>
  <head>
    <title>eegabrechnung API — Referenz</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>body { margin: 0; padding: 0; }</style>
  </head>
  <body>
    <redoc spec-url='data:application/json;base64,${specBase64}'></redoc>
    <script src="https://cdn.jsdelivr.net/npm/redoc/bundles/redoc.standalone.js"></script>
  </body>
</html>`;

// Ensure output directory exists
mkdirSync(dirname(OUT_HTML), { recursive: true });

// Write output
writeFileSync(OUT_HTML, html, 'utf-8');

console.log(`Generated: ${OUT_HTML}`);
console.log(`HTML size: ${(html.length / 1024).toFixed(1)} KB`);
