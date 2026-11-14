import { registerIcons, type IconData } from '@facile/muse';
import mdi from './mdi.json';
import solar from './solar.json';

type Collection = {
	prefix: string;
	width?: number;
	height?: number;
	icons: Record<string, { body: string; width?: number; height?: number }>;
	aliases?: Record<string, { parent: string }>;
};

function entries(collection: Collection): [string, IconData][] {
	const size = (icon: { width?: number; height?: number }) => ({
		width: icon.width ?? collection.width ?? 24,
		height: icon.height ?? collection.height ?? 24
	});
	const own = Object.entries(collection.icons).map(
		([name, icon]): [string, IconData] => [`${collection.prefix}:${name}`, { body: icon.body, ...size(icon) }]
	);
	const aliases = Object.entries(collection.aliases ?? {}).flatMap(([name, alias]): [string, IconData][] => {
		const parent = collection.icons[alias.parent];
		return parent ? [[`${collection.prefix}:${name}`, { body: parent.body, ...size(parent) }]] : [];
	});
	return [...own, ...aliases];
}

/** Hands the bundled Solar and MDI glyphs to muse's `Icon`, so it draws them without a network fetch. */
export function registerBundledIcons() {
	registerIcons(Object.fromEntries([...entries(solar as Collection), ...entries(mdi as Collection)]));
}
