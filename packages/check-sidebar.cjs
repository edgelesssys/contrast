// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// check-sidebar.js
const path = require('path');
const fs = require('fs');

// Get the file path from the command line arguments
const fileArg = process.argv[2];

if (!fileArg) {
    console.error('Usage: node read-data.js <path-to-sidebars.js>');
    process.exit(1);
}


function getDocIds(obj) {
    const foundIds = new Set();

    function walk(obj) {
        if (Array.isArray(obj)) {
            for (const item of obj) {
                walk(item);
            }
        } else if (obj && typeof obj === 'object') {
            if ('id' in obj) {
                const p = obj.id + ".md";
                if (foundIds.has(p)) {
                    console.error(`❌ Duplicate id found: "${obj.id}"`);
                    process.exit(1);
                }
                foundIds.add(p);
            }
            if ('items' in obj && Array.isArray(obj.items)) {
                walk(obj.items);
            }
        }
    }

    walk(obj);

    return foundIds;
}

function getMarkdownFiles(dir) {
    let results = [];

    const entries = fs.readdirSync(dir, { withFileTypes: true });

    for (const entry of entries) {
        const fullPath = path.join(dir, entry.name);

        if (entry.isDirectory()) {
            results = results.concat(getMarkdownFiles(fullPath));
        } else if (entry.isFile() && entry.name.endsWith('.md')) {
            results.push(fullPath);
        }
    }

    return results;
}


const sidebars = require(path.resolve(process.cwd(), fileArg));

const docIds = getDocIds(sidebars.docs);

const prefix = path.join(path.dirname(fileArg), "docs");
const mdFiles = new Set(getMarkdownFiles(prefix).map(str =>
    str.startsWith(prefix) ? str.slice(prefix.length + 1) : str
));

const missingFiles = [...docIds].filter(id => !mdFiles.has(id));
if (missingFiles.length > 0) {
    console.error('❌ Missing markdown files for IDs:');
    missingFiles.forEach(id => console.error(`  - ${id}`));
}

const extraFiles = [...mdFiles].filter(id => !docIds.has(id));
if (extraFiles.length > 0) {
    console.error('❌ Markdown files with no matching ID in data:');
    extraFiles.forEach(id => console.error(`  - ${id}`));
}

if (missingFiles.length > 0 || extraFiles.length > 0) {
    process.exit(1);
} else {
    console.log('✅ All IDs have corresponding .md files and vice versa.');
}
