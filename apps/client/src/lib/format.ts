/*
 * Formatters shared by every page that shows a number or a date. They live here because
 * "2 h 14 min" was being spelled three different ways across three routes, and a duration
 * that reads differently on the incidents page than on the monitor it belongs to reads as
 * two different durations. Every one of them goes through Intl in the session locale.
 */

import type { Bucket, MonitorStatus } from '$lib/backend';
import { locale, t } from '$lib/i18n/index.svelte';

const MISSING = '—';

/* A percentage nobody measured is not 100. Every uptime field on the API is
   nullable for that reason, and rendering null as a number throws the fact
   away. The API sends 99.5 for 99.5%, so the value is divided before Intl sees it. */
export function percent(value: number | null | undefined, digits = 2): string {
	if (value === null || value === undefined) return MISSING;
	return new Intl.NumberFormat(locale(), {
		style: 'percent',
		minimumFractionDigits: digits,
		maximumFractionDigits: digits
	}).format(value / 100);
}

function unit(value: number, name: Intl.NumberFormatOptions['unit'], digits = 0): string {
	return new Intl.NumberFormat(locale(), {
		style: 'unit',
		unit: name,
		unitDisplay: 'narrow',
		maximumFractionDigits: digits
	}).format(value);
}

export function milliseconds(value: number | null | undefined): string {
	if (value === null || value === undefined) return MISSING;
	if (value >= 1000) return unit(value / 1000, 'second', 2);
	return unit(Math.round(value), 'millisecond');
}

/* Coarse on purpose: an outage measured to the second reads as a stopwatch,
   and nobody acts differently on 4 h 12 min 09 s than on 4 h 12 min. */
export function duration(seconds: number | null | undefined): string {
	if (seconds === null || seconds === undefined) return MISSING;
	const total = Math.max(0, Math.round(seconds));
	if (total < 60) return unit(total, 'second');
	const minutes = Math.floor(total / 60);
	if (minutes < 60) return unit(minutes, 'minute');
	const hours = Math.floor(minutes / 60);
	if (hours < 24) return `${unit(hours, 'hour')} ${unit(minutes % 60, 'minute')}`;
	return `${unit(Math.floor(hours / 24), 'day')} ${unit(hours % 24, 'hour')}`;
}

/* The age of a moment, in the coarsest unit that is not zero. */
export function since(iso: string | null | undefined): string {
	if (!iso) return t('common.never');
	const seconds = Math.round((Date.now() - new Date(iso).getTime()) / 1000);
	if (seconds < 45) return t('common.justNow');
	const format = new Intl.RelativeTimeFormat(locale(), { numeric: 'auto' });
	if (seconds < 3600) return format.format(-Math.round(seconds / 60), 'minute');
	if (seconds < 86400) return format.format(-Math.round(seconds / 3600), 'hour');
	return format.format(-Math.round(seconds / 86400), 'day');
}

export function shortTime(iso: string): string {
	return new Intl.DateTimeFormat(locale(), { hour: '2-digit', minute: '2-digit' }).format(new Date(iso));
}

export function shortDay(iso: string): string {
	return new Intl.DateTimeFormat(locale(), { day: 'numeric', month: 'short' }).format(new Date(iso));
}

export function dateTime(iso: string): string {
	return new Intl.DateTimeFormat(locale(), { dateStyle: 'short', timeStyle: 'medium' }).format(new Date(iso));
}

export function date(iso: string): string {
	return new Intl.DateTimeFormat(locale(), { dateStyle: 'medium' }).format(new Date(iso));
}

export function clock(value: Date): string {
	return new Intl.DateTimeFormat(locale(), { timeStyle: 'medium' }).format(value);
}

/* The tone vocabulary is muse's (CHARTE §2), not a private one, so a status
   dot, a badge and a chart legend entry for the same monitor agree. */
export const STATUS_TONE: Record<MonitorStatus, 'success' | 'danger' | 'neutral' | 'warning'> = {
	up: 'success',
	down: 'danger',
	paused: 'neutral',
	unknown: 'warning'
};

export function statusLabel(status: MonitorStatus): string {
	return t(`monitor.status.${status}`);
}

/* A bucket with no checks is a gap in coverage, and a chart that plots it as
   zero invents an outage. LineChart has no notion of a hole, so the series
   carries the last measured value forward and the uptime strip — which does
   distinguish them — is what tells the truth about coverage. */
export function latencySeries(buckets: Bucket[]): number[] {
	let last = 0;
	return buckets.map((bucket) => {
		if (bucket.total > 0) last = bucket.avg_latency_ms;
		return last;
	});
}
