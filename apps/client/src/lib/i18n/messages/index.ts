import type { Catalogs, Message } from '@facile/muse';
import { incidents } from './incidents';
import { landing } from './landing';
import { maintenance } from './maintenance';
import { maintenanceScan } from './maintenance-scan';
import { monitorForms } from './monitor-forms';
import { monitors } from './monitors';
import { settings } from './settings';
import { settingsBus } from './settings-bus';
import { settingsWebhooks } from './settings-webhooks';
import { shell } from './shell';
import { status } from './status';
import type { Row } from '../row';

const rows = { ...shell, ...landing, ...monitors, ...monitorForms, ...incidents, ...status, ...settings, ...settingsBus, ...settingsWebhooks, ...maintenance, ...maintenanceScan };

export type AppKey = keyof typeof rows;
type AppCatalog = Record<AppKey, Message>;

function column(index: number): AppCatalog {
	const entries = Object.entries(rows as Record<string, Row>).map(([key, row]) => [key, row[index]]);
	return Object.fromEntries(entries) as AppCatalog;
}

/** The app's catalogs: English is the typed base, and every row carries all four locales. */
export const catalogs: Catalogs<AppCatalog> = {
	en: column(0),
	fr: column(1),
	es: column(2),
	de: column(3)
};
