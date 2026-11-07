/**
 * Regenerates `src/lib/icons/*.json` — the icon glyphs Sonde ships with.
 *
 * Run it after adding an icon anywhere in this app or after bumping `@facile/muse`:
 *
 *     bun run icons
 *
 * Why this exists: `iconify-icon` resolves unknown icons by calling `api.iconify.design` from
 * the user's browser. That makes every icon in a self-hosted app a third-party runtime
 * dependency — no icons on an air-gapped instance, a hard dependency on someone else's uptime,
 * and a request to a third party naming the glyphs (so, roughly, the page) each user is
 * looking at. Registering a collection up front means the API is never consulted.
 *
 * It scans *source*, not the build, so it does not need a build to have run — and it scans
 * muse too, because muse's own components carry the icon strings for the chrome they render.
 */
import { readdir, readFile, mkdir, writeFile } from 'node:fs/promises';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const clientRoot = join(here, '..');
const OUT_DIR = join(clientRoot, 'src/lib/icons');

/* Both trees are scanned: ours for app glyphs, muse's for the ones its components hardcode. */
const SCAN = [join(clientRoot, 'src'), join(clientRoot, 'node_modules/@facile/muse/src')];

const ICON_PATTERN = /\b(solar|mdi):[a-z0-9-]+/g;
const SCAN_EXTENSIONS = ['.svelte', '.ts', '.js'];

async function walk(dir: string): Promise<string[]> {
	const entries = await readdir(dir, { withFileTypes: true }).catch(() => []);
	const files = await Promise.all(
		entries.map((entry) => {
			const full = join(dir, entry.name);
			if (entry.isDirectory()) return walk(full);
			return SCAN_EXTENSIONS.some((ext) => entry.name.endsWith(ext)) ? [full] : [];
		})
	);
	return files.flat();
}

const found = new Set<string>();
for (const root of SCAN) {
	for (const file of await walk(root)) {
		const source = await readFile(file, 'utf8');
		for (const match of source.matchAll(ICON_PATTERN)) found.add(match[0]);
	}
}

const byPrefix = new Map<string, string[]>();
for (const icon of [...found].sort()) {
	const [prefix, name] = icon.split(':');
	byPrefix.set(prefix, [...(byPrefix.get(prefix) ?? []), name]);
}

await mkdir(OUT_DIR, { recursive: true });

for (const [prefix, names] of byPrefix) {
	const url = `https://api.iconify.design/${prefix}.json?icons=${names.join(',')}`;
	const response = await fetch(url);
	if (!response.ok) throw new Error(`${prefix}: ${response.status} ${response.statusText}`);

	const data = (await response.json()) as { icons?: Record<string, unknown>; aliases?: Record<string, unknown>; not_found?: string[] };
	/* A missing glyph must fail the generator, not fall through to a silent runtime fetch of
	   the one icon that is not bundled — that would put the CDN dependency back for good. */
	if (data.not_found?.length) throw new Error(`${prefix}: not found — ${data.not_found.join(', ')}`);

	const served = { ...data.icons, ...data.aliases };
	const missing = names.filter((name) => !(name in served));
	if (missing.length) throw new Error(`${prefix}: asked for ${names.length}, missing ${missing.join(', ')}`);
	const count = Object.keys(data.icons ?? {}).length;

	await writeFile(join(OUT_DIR, `${prefix}.json`), `${JSON.stringify(data)}\n`);
	console.log(`${prefix}: ${count} icons`);
}

console.log(`\nWrote ${byPrefix.size} collection(s) to src/lib/icons/`);
