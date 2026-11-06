import type { Row } from '../row';

/** The incidents page. */
export const incidents = {
	'incidents.description': [
		'Everything that went wrong, newest first.',
		'Tout ce qui a mal tourné, du plus récent au plus ancien.',
		'Todo lo que ha salido mal, lo más reciente primero.',
		'Alles, was schiefgelaufen ist, das Neueste zuerst.'
	],
	'incidents.range.7': ['7 days', '7 jours', '7 días', '7 Tage'],
	'incidents.range.30': ['30 days', '30 jours', '30 días', '30 Tage'],
	'incidents.range.90': ['90 days', '90 jours', '90 días', '90 Tage'],
	'incidents.window': ['Incident window', 'Période des incidents', 'Periodo de incidentes', 'Vorfallzeitraum'],
	'incidents.loadError': [
		'Could not load incidents',
		'Impossible de charger les incidents',
		'No se pudieron cargar los incidentes',
		'Vorfälle konnten nicht geladen werden'
	],
	'incidents.loadFailed': [
		'Failed to load incidents.',
		'Échec du chargement des incidents.',
		'Error al cargar los incidentes.',
		'Vorfälle konnten nicht geladen werden.'
	],
	'incidents.statsFailed': [
		'Failed to load incident stats.',
		'Échec du chargement des statistiques.',
		'Error al cargar las estadísticas de incidentes.',
		'Vorfallstatistiken konnten nicht geladen werden.'
	],
	'incidents.stat.openNow': ['Open now', 'Ouverts maintenant', 'Abiertos ahora', 'Jetzt offen'],
	'incidents.stat.nothingOngoing': ['Nothing ongoing', 'Rien en cours', 'Nada en curso', 'Nichts offen'],
	'incidents.stat.unresolved': ['Still unresolved', 'Pas encore résolus', 'Aún sin resolver', 'Noch ungelöst'],
	'incidents.stat.total': [
		'Incidents, {days} days',
		'Incidents, {days} jours',
		'Incidentes, {days} días',
		'Vorfälle, {days} Tage'
	],
	'incidents.stat.resolved': ['{count} resolved', '{count} résolus', '{count} resueltos', '{count} gelöst'],
	'incidents.stat.mttr': ['MTTR', 'MTTR', 'MTTR', 'MTTR'],
	'incidents.stat.mttrHint': [
		'Resolved incidents only: an open one has no repair time yet',
		'Incidents résolus uniquement : un incident ouvert n’a pas encore de durée de réparation',
		'Solo incidentes resueltos: uno abierto aún no tiene tiempo de reparación',
		'Nur gelöste Vorfälle: ein offener hat noch keine Reparaturzeit'
	],
	'incidents.stat.downtime': ['Total downtime', 'Indisponibilité totale', 'Tiempo total de caída', 'Gesamte Ausfallzeit'],
	'incidents.stat.noOutage': [
		'No outage recorded',
		'Aucune panne enregistrée',
		'Sin caídas registradas',
		'Kein Ausfall aufgezeichnet'
	],
	'incidents.stat.longest': ['Longest {duration}', 'Plus longue : {duration}', 'Más larga: {duration}', 'Längster: {duration}'],
	'incidents.overTime': [
		'Incidents over time',
		'Incidents dans le temps',
		'Incidentes en el tiempo',
		'Vorfälle im Zeitverlauf'
	],
	'incidents.perDay': [
		'Incidents opened per day',
		'Incidents ouverts par jour',
		'Incidentes abiertos por día',
		'Pro Tag eröffnete Vorfälle'
	],
	'incidents.series.opened': ['Opened', 'Ouverts', 'Abiertos', 'Eröffnet'],
	'incidents.noneInWindow': [
		'No incidents in this window',
		'Aucun incident sur cette période',
		'Sin incidentes en este periodo',
		'Keine Vorfälle in diesem Zeitraum'
	],
	'incidents.pain': ['Where the pain is', 'Où ça fait mal', 'Dónde duele', 'Wo es wehtut'],
	'incidents.byMonitor': [
		'Downtime by monitor',
		'Indisponibilité par sonde',
		'Caídas por monitor',
		'Ausfallzeit pro Monitor'
	],
	'incidents.other': ['Other', 'Autres', 'Otros', 'Sonstige'],
	'incidents.downtimeLabel': ['Downtime', 'Indisponibilité', 'Caída', 'Ausfallzeit'],
	'incidents.all': ['All incidents', 'Tous les incidents', 'Todos los incidentes', 'Alle Vorfälle'],
	'incidents.empty.title': ['No incidents', 'Aucun incident', 'Sin incidentes', 'Keine Vorfälle'],
	'incidents.empty.description': [
		'When a monitor fails three checks in a row, it shows up here.',
		'Quand une sonde échoue trois vérifications de suite, elle apparaît ici.',
		'Cuando un monitor falla tres comprobaciones seguidas, aparece aquí.',
		'Wenn ein Monitor drei Prüfungen in Folge nicht besteht, erscheint er hier.'
	],
	'incidents.col.state': ['State', 'État', 'Estado', 'Status'],
	'incidents.col.cause': ['Cause', 'Cause', 'Causa', 'Ursache'],
	'incidents.col.opened': ['Opened', 'Ouvert', 'Abierto', 'Eröffnet'],
	'incidents.col.resolved': ['Resolved', 'Résolu', 'Resuelto', 'Gelöst'],
	'incidents.col.lasted': ['Lasted', 'Durée', 'Duración', 'Dauer'],
	'incidents.state.resolved': ['resolved', 'résolu', 'resuelto', 'gelöst'],
	'incidents.state.ongoing': ['ongoing', 'en cours', 'en curso', 'laufend']
} satisfies Record<string, Row>;
