#!/usr/bin/env node
'use strict';

const fastGlob = require('fast-glob');
const {optimize, extendDefaultPlugins} = require('svgo');
const {resolve, parse} = require('path');
const {readFile, writeFile, mkdir} = require('fs').promises;

const glob = (pattern) => fastGlob.sync(pattern, {cwd: resolve(__dirname), absolute: true});
const outputDir = resolve(__dirname, '../public/img/svg');

function exit(err) {
  if (err) console.error(err);
  process.exit(err ? 1 : 0);
}

async function processFile(file, {prefix, fullName} = {}) {
  let name;

  if (fullName) {
    name = fullName;
  } else {
    name = parse(file).name;
    if (prefix) name = `${prefix}-${name}`;
    if (prefix === 'octicon') name = name.replace(/-[0-9]+$/, ''); // chop of '-16' on octicons
  }

  const {data} = optimize(await readFile(file, 'utf8'), {
    plugins: extendDefaultPlugins([
      'removeXMLNS',
      'removeDimensions',
      {
        name: 'addClassesToSVGElement',
        params: {
          classNames: [
            'svg',
            name,
          ],
        },
      },
      {
        name: 'addAttributesToSVGElement',
        params: {
          attributes: [
            {'width': '16'},
            {'height': '16'},
            {'aria-hidden': 'true'},
          ],
        },
      },
    ]),
  });
  await writeFile(resolve(outputDir, `${name}.svg`), data);
}

function processFiles(pattern, opts) {
  return glob(pattern).map((file) => processFile(file, opts));
}

async function main() {
  try {
    await mkdir(outputDir);
  } catch {}

  await Promise.all([
    ...processFiles('../node_modules/@primer/octicons/build/svg/*-16.svg', {prefix: 'octicon'}),
    ...processFiles('../web_src/svg/*.svg'),
    ...processFiles('../public/img/gitea.svg', {fullName: 'gitea-gitea'}),
  ]);
}

main().then(exit).catch(exit);

