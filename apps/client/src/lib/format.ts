/*
 * Formatters shared by every page that shows a number. They live here because
 * "2 h 14 min" was being spelled three different ways across three routes, and
 * a duration that reads differently on the incidents page than on the monitor
 * it belongs to reads as two different durations.
 */

import type { Bucket, MonitorStatus } from '$lib/backend';

/* A percentage nobody measured is not 100. Every uptime field on the API is
   nullable for that reason, and rendering null as a number throws the fact
   away. */
export function percent(value: number | null | undefined, digits = 2): string {
	if (value === null || value === undefined) return '—';
	return `${value.toFixed(digits)}%`;
}

export function milliseconds(value: number | null | undefined): string {
	if (value === null || value === undefined) return '—';
	if (value >= 1000) return `${(value / 1000).toFixed(2)} s`;
	return `${Math.round(value)} ms`;
}

/* Coarse on purpose: an outage measured to the second reads as a stopwatch,
   and nobody acts differently on 4 h 12 min 09 s than on 4 h 12 min. */
export function duration(seconds: number | null | undefined): string {
	if (seconds === null || seconds === undefined) return '—';
	const total = Math.max(0, Math.round(seconds));
	if (total < 60) return `${total} s`;
	const minutes = Math.floor(total / 60);
	if (minutes < 60) return `${minutes} min`;
	const hours = Math.floor(minutes / 60);
	if (hours < 24) return `${hours} h ${minutes % 60} min`;
	return `${Math.floor(hours / 24)} d ${hours % 24} h`;
}

export function since(iso: string | null | undefined): string {
	if (!iso) return 'never';
	const seconds = (Date.now() - new Date(iso).getTime()) / 1000;
	if (seconds < 45) return 'just now';
	return `${duration(seconds)} ago`;
}

export function shortTime(iso: string): string {
	return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

export function shortDay(iso: string): string {
	return new Date(iso).toLocaleDateString([], { day: 'numeric', month: 'short' });
}

/* The tone vocabulary is muse's (CHARTE §2), not a private one, so a status
   dot, a badge and a chart legend entry for the same monitor agree. */
export const STATUS_TONE: Record<MonitorStatus, 'success' | 'danger' | 'neutral' | 'warning'> = {
	up: 'success',
	down: 'danger',
	paused: 'neutral',
	unknown: 'warning'
};

export const STATUS_LABEL: Record<MonitorStatus, string> = {
	up: 'Up',
	down: 'Down',
	paused: 'Paused',
	unknown: 'No data'
};

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
