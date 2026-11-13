import { backend, type MaintenanceScan } from '$lib/backend';

const pollEveryMs = 2000;
const runningStates = ['queued', 'in_progress', 'delivering'];

/** Whether a scan is still going: queued on GitHub, running, or delivering its report. */
export function isRunning(scan: MaintenanceScan | null): boolean {
	return !!scan && runningStates.includes(scan.state);
}

function queuedNow(): MaintenanceScan {
	return {
		state: 'queued',
		conclusion: '',
		run_url: '',
		started_at: new Date().toISOString(),
		steps: [],
		progress: null,
		last_report_at: null,
		last_delivery: null
	};
}

/**
 * Polls the scan in flight every two seconds while it runs and the tab is visible, and
 * calls `onEnd` once with the final state of a scan it saw running. A hidden tab stops
 * the polling; showing it again picks up where it left off.
 */
export class ScanWatch {
	scan = $state<MaintenanceScan | null>(null);
	#timer: ReturnType<typeof setTimeout> | undefined;
	#watching = false;
	#onEnd: (scan: MaintenanceScan) => void;
	#onVisible = () => {
		if (!document.hidden && this.#watching) this.#schedule(0);
	};

	constructor(onEnd: (scan: MaintenanceScan) => void) {
		this.#onEnd = onEnd;
	}

	start() {
		document.addEventListener('visibilitychange', this.#onVisible);
		void this.#tick();
	}

	stop() {
		clearTimeout(this.#timer);
		document.removeEventListener('visibilitychange', this.#onVisible);
	}

	/** Shows the scan as queued at once, before GitHub has listed its run. */
	queued() {
		this.scan = queuedNow();
		this.#watching = true;
		this.#schedule(pollEveryMs);
	}

	#schedule(delay: number) {
		clearTimeout(this.#timer);
		this.#timer = setTimeout(() => void this.#tick(), delay);
	}

	async #tick() {
		if (document.hidden) return;
		const next = await backend.currentScan().catch(() => this.scan);
		if (!next) return;
		this.scan = next;
		if (isRunning(next)) {
			this.#watching = true;
			this.#schedule(pollEveryMs);
		} else if (this.#watching) {
			this.#watching = false;
			this.#onEnd(next);
		}
	}
}
