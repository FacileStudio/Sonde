import type { CliCommand } from '@facile/muse';
import { t } from '$lib/i18n/index.svelte';

export const RELEASES_URL = 'https://github.com/FacileStudio/sonde-cli/releases';
export const CHECKSUMS_URL = `${RELEASES_URL}/latest/download/checksums.txt`;

export const INSTALL_METHODS: CliCommand[] = [
	{
		get label() {
			return t('downloads.install.script');
		},
		command: 'curl -fsSL https://raw.githubusercontent.com/FacileStudio/sonde-cli/main/install.sh | bash'
	},
	{
		get label() {
			return t('downloads.install.facile');
		},
		command: 'facile install sonde'
	},
	{
		get label() {
			return t('downloads.install.brew');
		},
		command: 'brew install --cask FacileStudio/tap/sonde'
	}
];
