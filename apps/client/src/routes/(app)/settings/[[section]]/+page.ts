import { redirect } from '@sveltejs/kit';

export const prerender = false;

const SECTIONS = ['profile', 'preferences', 'notifications', 'bus', 'advanced'];

export function load({ params }) {
	const section = params.section;
	if (!section) return;
	if (section === 'appearance') redirect(307, '/settings/preferences');
	if (!SECTIONS.includes(section)) redirect(307, '/settings');
}
