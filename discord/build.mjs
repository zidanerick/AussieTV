import { build } from 'esbuild';
import { mkdir, copyFile } from 'node:fs/promises';
await mkdir('dist', {recursive:true});
await build({entryPoints:['app.mjs'],bundle:true,minify:true,format:'esm',platform:'browser',target:'es2022',outfile:'dist/app.js',legalComments:'inline'});
await Promise.all(['index.html','style.css'].map(name=>copyFile(name,'dist/'+name)));
