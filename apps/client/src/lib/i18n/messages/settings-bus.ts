import type { Row } from '../row';

/** Settings: the event bus and retention. */
export const settingsBus = {
	'bus.title': ['Antenne', 'Antenne', 'Antenne', 'Antenne'],
	'bus.description': [
		'Sonde publishes every monitor transition to the Antenne bus, and Antenne decides who hears about it. Sonde does not fan out to humans itself.',
		'Sonde publie chaque changement d’état d’une sonde sur le bus Antenne, et Antenne décide qui est prévenu. Sonde ne prévient pas les personnes elle-même.',
		'Sonde publica cada cambio de estado de un monitor en el bus Antenne, y Antenne decide a quién avisar. Sonde no avisa a las personas por sí misma.',
		'Sonde veröffentlicht jeden Statuswechsel eines Monitors auf dem Antenne-Bus, und Antenne entscheidet, wer es erfährt. Sonde benachrichtigt Menschen nicht selbst.'
	],
	'bus.loadError': [
		'Could not read the bus state',
		'Impossible de lire l’état du bus',
		'No se pudo leer el estado del bus',
		'Der Bus-Status konnte nicht gelesen werden'
	],
	'bus.loadFailed': [
		'Could not read this instance’s integrations.',
		'Impossible de lire les intégrations de cette instance.',
		'No se pudieron leer las integraciones de esta instancia.',
		'Die Integrationen dieser Instanz konnten nicht gelesen werden.'
	],
	'bus.state.connected': ['Connected', 'Connecté', 'Conectado', 'Verbunden'],
	'bus.state.connecting': ['Connecting', 'Connexion', 'Conectando', 'Verbindet'],
	'bus.state.reconnecting': ['Reconnecting', 'Reconnexion', 'Reconectando', 'Verbindet erneut'],
	'bus.state.disabled': ['Disabled', 'Désactivé', 'Desactivado', 'Deaktiviert'],
	'bus.connection': ['Connection', 'Connexion', 'Conexión', 'Verbindung'],
	'bus.connection.configured': [
		'Publishing runs in the background and retries on its own. A restart is never the fix.',
		'La publication tourne en arrière-plan et réessaie seule. Un redémarrage n’est jamais la solution.',
		'La publicación se ejecuta en segundo plano y reintenta sola. Reiniciar nunca es la solución.',
		'Die Veröffentlichung läuft im Hintergrund und versucht es selbst erneut. Ein Neustart ist nie die Lösung.'
	],
	'bus.connection.unconfigured': [
		'ANTENNE_URL is empty on this instance, so nothing is published. Set it, with ANTENNE_SECRET, and restart.',
		'ANTENNE_URL est vide sur cette instance : rien n’est publié. Renseignez-la avec ANTENNE_SECRET, puis redémarrez.',
		'ANTENNE_URL está vacío en esta instancia, así que no se publica nada. Defínelo junto con ANTENNE_SECRET y reinicia.',
		'ANTENNE_URL ist auf dieser Instanz leer, daher wird nichts veröffentlicht. Setzen Sie es zusammen mit ANTENNE_SECRET und starten Sie neu.'
	],
	'bus.instance': ['Instance', 'Instance', 'Instancia', 'Instanz'],
	'bus.appIdentity': ['App identity', 'Identité de l’application', 'Identidad de la aplicación', 'App-Identität'],
	'bus.appIdentityHint': [
		'The name events are attributed to on the bus.',
		'Le nom auquel les événements sont attribués sur le bus.',
		'El nombre al que se atribuyen los eventos en el bus.',
		'Der Name, dem Ereignisse auf dem Bus zugeordnet werden.'
	],
	'bus.channel.down': ['Outage channel', 'Canal des pannes', 'Canal de caídas', 'Ausfallkanal'],
	'bus.channel.up': ['Recovery channel', 'Canal des rétablissements', 'Canal de recuperación', 'Wiederherstellungskanal'],
	'bus.emitting': ['emitting', 'émet', 'emitiendo', 'sendet'],
	'bus.published': ['Events published', 'Événements publiés', 'Eventos publicados', 'Veröffentlichte Ereignisse'],
	'bus.publishedHint': [
		'Since this container started. A restart resets the count, not the delivery.',
		'Depuis le démarrage de ce conteneur. Un redémarrage remet le compteur à zéro, pas la livraison.',
		'Desde que arrancó este contenedor. Un reinicio pone a cero la cuenta, no la entrega.',
		'Seit dem Start dieses Containers. Ein Neustart setzt den Zähler zurück, nicht die Zustellung.'
	],
	'bus.pending': ['Events waiting', 'Événements en attente', 'Eventos en espera', 'Wartende Ereignisse'],
	'bus.pendingHint': [
		'Queued in the database while the bus is unreachable. They are sent in order once it answers.',
		'Mis en file dans la base tant que le bus est injoignable. Ils partent dans l’ordre dès qu’il répond.',
		'En cola en la base de datos mientras el bus es inalcanzable. Se envían en orden en cuanto responda.',
		'In der Datenbank eingereiht, solange der Bus nicht erreichbar ist. Sie werden der Reihe nach gesendet, sobald er antwortet.'
	],
	'bus.lastEvent': ['Last event', 'Dernier événement', 'Último evento', 'Letztes Ereignis'],
	'bus.lastEventHint': [
		'The most recent transition that left this box.',
		'Le changement d’état le plus récent parti de cette machine.',
		'La transición más reciente que salió de esta máquina.',
		'Der jüngste Statuswechsel, der diese Maschine verlassen hat.'
	],
	'bus.lastError': [
		'Last error from the bus',
		'Dernière erreur du bus',
		'Último error del bus',
		'Letzter Fehler des Bus'
	],

	'retention.title': ['History', 'Historique', 'Historial', 'Verlauf'],
	'retention.description': [
		'How much of the past this instance can still answer questions about.',
		'Jusqu’où cette instance peut encore répondre sur le passé.',
		'Cuánto del pasado puede responder todavía esta instancia.',
		'Wie weit in die Vergangenheit diese Instanz noch Fragen beantworten kann.'
	],
	'retention.raw': ['Raw checks', 'Vérifications brutes', 'Comprobaciones sin procesar', 'Rohprüfungen'],
	'retention.rawHint': [
		'Every individual probe. Set with SONDE_RAW_RETENTION_DAYS.',
		'Chaque mesure individuelle. Réglé par SONDE_RAW_RETENTION_DAYS.',
		'Cada medición individual. Se define con SONDE_RAW_RETENTION_DAYS.',
		'Jede einzelne Messung. Eingestellt über SONDE_RAW_RETENTION_DAYS.'
	],
	'retention.days': [
		{ one: '{count} day', other: '{count} days' },
		{ one: '{count} jour', other: '{count} jours' },
		{ one: '{count} día', other: '{count} días' },
		{ one: '{count} Tag', other: '{count} Tage' }
	],
	'retention.hourly': ['Hourly rollups', 'Agrégats horaires', 'Agregados por hora', 'Stundenaggregate'],
	'retention.hourlyHint': [
		'What every uptime percentage and chart is read from, so this is the real limit on how far back a window can go.',
		'La source de chaque pourcentage et graphique de disponibilité : c’est donc la vraie limite de recul d’une période.',
		'De donde se lee cada porcentaje y gráfico de disponibilidad, así que es el límite real de hasta dónde puede llegar un periodo.',
		'Daraus werden alle Verfügbarkeitswerte und Diagramme gelesen; das ist also die tatsächliche Grenze, wie weit ein Zeitraum zurückreicht.'
	],
	'retention.months': [
		{ one: '{count} month', other: '{count} months' },
		{ one: '{count} mois', other: '{count} mois' },
		{ one: '{count} mes', other: '{count} meses' },
		{ one: '{count} Monat', other: '{count} Monate' }
	],
	'retention.threshold': ['Failure threshold', 'Seuil d’échec', 'Umbral de fallos', 'Fehlerschwelle'],
	'retention.thresholdHint': [
		'Consecutive failed checks before an incident opens and the bus hears about it.',
		'Nombre de vérifications échouées consécutives avant l’ouverture d’un incident et l’alerte du bus.',
		'Comprobaciones fallidas consecutivas antes de abrir un incidente y avisar al bus.',
		'Aufeinanderfolgende fehlgeschlagene Prüfungen, bevor ein Vorfall eröffnet und der Bus informiert wird.'
	],
	'retention.logs': ['Log shipping', 'Envoi des journaux', 'Envío de registros', 'Log-Weiterleitung'],
	'retention.logsLocal': [
		'JOURNAL_URL is empty, so logs stay in this container.',
		'JOURNAL_URL est vide : les journaux restent dans ce conteneur.',
		'JOURNAL_URL está vacío, así que los registros se quedan en este contenedor.',
		'JOURNAL_URL ist leer, daher bleiben die Logs in diesem Container.'
	],
	'retention.journal': ['Journal', 'Journal', 'Journal', 'Journal'],
	'retention.localOnly': ['Local only', 'Local uniquement', 'Solo local', 'Nur lokal']
} satisfies Record<string, Row>;
