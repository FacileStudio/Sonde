import type { Row } from '../row';

/** The monitor list and the monitor page. */
export const monitors = {
	'monitors.description': [
		'What Sonde watches, and whether it is alive right now.',
		'Ce que Sonde surveille, et si cela répond en ce moment.',
		'Lo que vigila Sonde y si responde ahora mismo.',
		'Was Sonde überwacht und ob es gerade antwortet.'
	],
	'monitors.new': ['New monitor', 'Nouvelle sonde', 'Nuevo monitor', 'Neuer Monitor'],
	'monitors.loadError': [
		'Could not load monitors',
		'Impossible de charger les sondes',
		'No se pudieron cargar los monitores',
		'Monitore konnten nicht geladen werden'
	],
	'monitors.loadFailed': [
		'Failed to load monitors.',
		'Échec du chargement des sondes.',
		'Error al cargar los monitores.',
		'Monitore konnten nicht geladen werden.'
	],
	'monitors.empty.title': ['No monitors yet', 'Aucune sonde pour l’instant', 'Aún no hay monitores', 'Noch keine Monitore'],
	'monitors.empty.description': [
		'Add an HTTP endpoint, a TCP port, or a push heartbeat to start watching.',
		'Ajoutez un point d’accès HTTP, un port TCP ou un signal push pour commencer la surveillance.',
		'Añade un endpoint HTTP, un puerto TCP o un latido push para empezar a vigilar.',
		'Fügen Sie einen HTTP-Endpunkt, einen TCP-Port oder einen Push-Herzschlag hinzu, um die Überwachung zu starten.'
	],
	'monitors.empty.cta': [
		'Create your first monitor',
		'Créer votre première sonde',
		'Crea tu primer monitor',
		'Ersten Monitor erstellen'
	],
	'monitors.stat.up': ['Up', 'En ligne', 'En línea', 'Online'],
	'monitors.stat.ofMonitors': [
		'of {count} monitors',
		'sur {count} sondes',
		'de {count} monitores',
		'von {count} Monitoren'
	],
	'monitors.stat.down': ['Down', 'Hors ligne', 'Caídos', 'Offline'],
	'monitors.stat.openIncidents': [
		{ one: '{count} open incident', other: '{count} open incidents' },
		{ one: '{count} incident ouvert', other: '{count} incidents ouverts' },
		{ one: '{count} incidente abierto', other: '{count} incidentes abiertos' },
		{ one: '{count} offener Vorfall', other: '{count} offene Vorfälle' }
	],
	'monitors.stat.noOpenIncidents': [
		'no open incidents',
		'aucun incident ouvert',
		'ningún incidente abierto',
		'keine offenen Vorfälle'
	],
	'monitors.stat.uptime24': ['Uptime 24 h', 'Disponibilité 24 h', 'Disponibilidad 24 h', 'Verfügbarkeit 24 h'],
	'monitors.stat.checks': [
		{ one: '{count} check', other: '{count} checks' },
		{ one: '{count} vérification', other: '{count} vérifications' },
		{ one: '{count} comprobación', other: '{count} comprobaciones' },
		{ one: '{count} Prüfung', other: '{count} Prüfungen' }
	],
	'monitors.stat.latency24': [
		'Avg latency 24 h',
		'Latence moy. 24 h',
		'Latencia media 24 h',
		'Ø Latenz 24 h'
	],
	'monitors.stat.noProbes': ['No probes yet', 'Aucune mesure', 'Aún sin mediciones', 'Noch keine Messungen'],
	'monitors.col.monitor': ['Monitor', 'Sonde', 'Monitor', 'Monitor'],
	'monitors.col.last24': ['Last 24 h', '24 dernières heures', 'Últimas 24 h', 'Letzte 24 h'],
	'monitors.col.uptime': ['Uptime', 'Disponibilité', 'Disponibilidad', 'Verfügbarkeit'],
	'monitors.col.latency': ['Latency', 'Latence', 'Latencia', 'Latenz'],
	'monitors.col.checked': ['Checked', 'Vérifiée', 'Comprobado', 'Geprüft'],
	'monitors.openBadge': ['{count} open', '{count} ouvert(s)', '{count} abierto(s)', '{count} offen'],
	'monitors.uptimeLabel': [
		'{name} uptime, last 24 hours',
		'Disponibilité de {name}, 24 dernières heures',
		'Disponibilidad de {name}, últimas 24 horas',
		'Verfügbarkeit von {name}, letzte 24 Stunden'
	],
	'monitors.openAria': ['Open {name}', 'Ouvrir {name}', 'Abrir {name}', '{name} öffnen'],
	'monitors.deleteAria': ['Delete {name}', 'Supprimer {name}', 'Eliminar {name}', '{name} löschen'],
	'monitors.created': [
		'{name} is being watched.',
		'{name} est sous surveillance.',
		'{name} está bajo vigilancia.',
		'{name} wird jetzt überwacht.'
	],
	'monitors.deleted': ['{name} deleted.', '{name} supprimée.', '{name} eliminado.', '{name} gelöscht.'],
	'monitors.deleteFailed': [
		'Could not delete the monitor.',
		'Impossible de supprimer la sonde.',
		'No se pudo eliminar el monitor.',
		'Der Monitor konnte nicht gelöscht werden.'
	],
	'monitors.delete.title': ['Delete {name}?', 'Supprimer {name} ?', '¿Eliminar {name}?', '{name} löschen?'],
	'monitors.delete.titleFallback': [
		'Delete monitor?',
		'Supprimer la sonde ?',
		'¿Eliminar el monitor?',
		'Monitor löschen?'
	],
	'monitors.delete.description': [
		'Its check history, uptime record and incidents are deleted with it. This cannot be undone.',
		'Son historique de vérifications, sa disponibilité et ses incidents sont supprimés avec elle. Cette action est irréversible.',
		'Su historial de comprobaciones, su disponibilidad y sus incidentes se eliminan con él. No se puede deshacer.',
		'Prüfverlauf, Verfügbarkeitsdaten und Vorfälle werden mit gelöscht. Das lässt sich nicht rückgängig machen.'
	],
	'monitors.delete.confirm': ['Delete monitor', 'Supprimer la sonde', 'Eliminar monitor', 'Monitor löschen'],

	'monitor.status.up': ['Up', 'En ligne', 'En línea', 'Online'],
	'monitor.status.down': ['Down', 'Hors ligne', 'Caído', 'Offline'],
	'monitor.status.paused': ['Paused', 'En pause', 'En pausa', 'Pausiert'],
	'monitor.status.unknown': ['No data', 'Aucune donnée', 'Sin datos', 'Keine Daten'],
	'monitor.title': ['Monitor', 'Sonde', 'Monitor', 'Monitor'],
	'monitor.loadFailed': [
		'Failed to load monitor.',
		'Échec du chargement de la sonde.',
		'Error al cargar el monitor.',
		'Der Monitor konnte nicht geladen werden.'
	],
	'monitor.loadError': [
		'Could not load this monitor',
		'Impossible de charger cette sonde',
		'No se pudo cargar este monitor',
		'Dieser Monitor konnte nicht geladen werden'
	],
	'monitor.pushHeartbeat': ['Push heartbeat', 'Signal push', 'Latido push', 'Push-Herzschlag'],
	'monitor.watching': ['Watching', 'Sous surveillance', 'Vigilando', 'Wird überwacht'],
	'monitor.delete': ['Delete', 'Supprimer', 'Eliminar', 'Löschen'],
	'monitor.somethingWrong': [
		'Something went wrong',
		'Une erreur est survenue',
		'Algo ha salido mal',
		'Etwas ist schiefgelaufen'
	],
	'monitor.window.24h': [
		'Uptime, 24 hours',
		'Disponibilité, 24 heures',
		'Disponibilidad, 24 horas',
		'Verfügbarkeit, 24 Stunden'
	],
	'monitor.window.7d': [
		'Uptime, 7 days',
		'Disponibilité, 7 jours',
		'Disponibilidad, 7 días',
		'Verfügbarkeit, 7 Tage'
	],
	'monitor.window.30d': [
		'Uptime, 30 days',
		'Disponibilité, 30 jours',
		'Disponibilidad, 30 días',
		'Verfügbarkeit, 30 Tage'
	],
	'monitor.window.90d': [
		'Uptime, 90 days',
		'Disponibilité, 90 jours',
		'Disponibilidad, 90 días',
		'Verfügbarkeit, 90 Tage'
	],
	'monitor.noChecks': [
		'No checks recorded yet',
		'Aucune vérification enregistrée',
		'Aún no hay comprobaciones registradas',
		'Noch keine Prüfungen aufgezeichnet'
	],
	'monitor.failedOf': [
		'{failed} failed of {total} checks',
		'{failed} échouées sur {total} vérifications',
		'{failed} fallidas de {total} comprobaciones',
		'{failed} von {total} Prüfungen fehlgeschlagen'
	],
	'monitor.heartbeat.title': ['Heartbeat', 'Signal de vie', 'Latido', 'Herzschlag'],
	'monitor.heartbeat.description': [
		'Have the job POST this URL every interval. A GET answers 200 without recording anything, so the method matters. Silence past the interval opens an incident.',
		'Faites envoyer un POST à cette URL par la tâche à chaque intervalle. Un GET répond 200 sans rien enregistrer : la méthode compte. Un silence au-delà de l’intervalle ouvre un incident.',
		'Haz que la tarea envíe un POST a esta URL en cada intervalo. Un GET responde 200 sin registrar nada, así que el método importa. El silencio más allá del intervalo abre un incidente.',
		'Lassen Sie den Job in jedem Intervall einen POST an diese URL senden. Ein GET antwortet mit 200, ohne etwas aufzuzeichnen, die Methode zählt also. Schweigen über das Intervall hinaus eröffnet einen Vorfall.'
	],
	'monitor.heartbeat.url': ['Heartbeat URL', 'URL du signal de vie', 'URL del latido', 'Herzschlag-URL'],
	'monitor.heartbeat.cron': ['Cron command', 'Commande cron', 'Comando cron', 'Cron-Befehl'],
	'monitor.settings': ['Settings', 'Réglages', 'Ajustes', 'Einstellungen'],
	'monitor.notSaved': ['Not saved', 'Non enregistré', 'No guardado', 'Nicht gespeichert'],
	'monitor.saved': ['Monitor saved.', 'Sonde enregistrée.', 'Monitor guardado.', 'Monitor gespeichert.'],
	'monitor.saveFailed': [
		'Failed to save the monitor.',
		'Échec de l’enregistrement de la sonde.',
		'Error al guardar el monitor.',
		'Der Monitor konnte nicht gespeichert werden.'
	],
	'monitor.noIncidents': [
		'Nothing has gone wrong on this monitor yet.',
		'Rien n’a encore échoué sur cette sonde.',
		'Aún no ha fallado nada en este monitor.',
		'Bei diesem Monitor ist noch nichts schiefgegangen.'
	],
	'monitor.incident.closed': ['down', 'hors ligne', 'caído', 'offline'],
	'monitor.incident.open': [
		'down, still open',
		'hors ligne, toujours ouvert',
		'caído, sigue abierto',
		'offline, noch offen'
	]
} satisfies Record<string, Row>;
