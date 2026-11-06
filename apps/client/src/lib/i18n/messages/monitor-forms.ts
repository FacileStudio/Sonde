import type { Row } from '../row';

/** The monitor forms, the history charts and the uptime strip. */
export const monitorForms = {
	'form.types.http': [
		'HTTP(S): probe a URL for status code and keyword',
		'HTTP(S) : teste une URL (code de statut, mot-clé)',
		'HTTP(S): comprueba una URL (código de estado y palabra clave)',
		'HTTP(S): prüft eine URL auf Statuscode und Stichwort'
	],
	'form.types.tcp': [
		'TCP: check that a host accepts connections',
		'TCP : vérifie qu’un hôte accepte les connexions',
		'TCP: comprueba que un host acepta conexiones',
		'TCP: prüft, ob ein Host Verbindungen annimmt'
	],
	'form.types.push': [
		'Push: a cron job pings Sonde, silence means down',
		'Push : une tâche cron appelle Sonde, le silence signifie une panne',
		'Push: una tarea cron avisa a Sonde, el silencio significa caída',
		'Push: ein Cron-Job meldet sich bei Sonde, Schweigen bedeutet Ausfall'
	],
	'form.namePlaceholder': ['Marketing site', 'Site vitrine', 'Sitio de marketing', 'Marketing-Website'],
	'form.slugHelper': [
		'Identifies the monitor on public status pages.',
		'Identifie la sonde sur les pages d’état publiques.',
		'Identifica el monitor en las páginas de estado públicas.',
		'Kennzeichnet den Monitor auf öffentlichen Statusseiten.'
	],
	'form.url': ['URL', 'URL', 'URL', 'URL'],
	'form.hostPort': ['Host and port', 'Hôte et port', 'Host y puerto', 'Host und Port'],
	'form.expectedStatus': ['Expected status', 'Statut attendu', 'Estado esperado', 'Erwarteter Status'],
	'form.expectedKeyword': ['Expected keyword', 'Mot-clé attendu', 'Palabra clave esperada', 'Erwartetes Stichwort'],
	'form.keywordHelper': [
		'Optional. The body must contain it.',
		'Facultatif. Le corps de la réponse doit le contenir.',
		'Opcional. El cuerpo debe contenerla.',
		'Optional. Der Antworttext muss es enthalten.'
	],
	'form.interval': ['Interval (seconds)', 'Intervalle (secondes)', 'Intervalo (segundos)', 'Intervall (Sekunden)'],
	'form.intervalHelper': ['At least 20.', '20 au minimum.', 'Mínimo 20.', 'Mindestens 20.'],
	'form.timeout': ['Timeout (seconds)', 'Délai d’attente (secondes)', 'Tiempo de espera (segundos)', 'Zeitlimit (Sekunden)'],
	'form.timeoutHelper': [
		'Must be shorter than the interval.',
		'Doit être plus court que l’intervalle.',
		'Debe ser menor que el intervalo.',
		'Muss kürzer als das Intervall sein.'
	],
	'form.enabled': ['Checking enabled', 'Vérification activée', 'Comprobación activada', 'Prüfung aktiviert'],

	'drawer.types.http': [
		'HTTP(S): probe a URL',
		'HTTP(S) : teste une URL',
		'HTTP(S): comprueba una URL',
		'HTTP(S): prüft eine URL'
	],
	'drawer.types.tcp': [
		'TCP: check a port accepts connections',
		'TCP : vérifie qu’un port accepte les connexions',
		'TCP: comprueba que un puerto acepta conexiones',
		'TCP: prüft, ob ein Port Verbindungen annimmt'
	],
	'drawer.types.push': [
		'Push: a cron job pings Sonde',
		'Push : une tâche cron appelle Sonde',
		'Push: una tarea cron avisa a Sonde',
		'Push: ein Cron-Job meldet sich bei Sonde'
	],
	'drawer.description': [
		'Checks start on the next tick, every 60 seconds. Tune the rest on the monitor itself.',
		'Les vérifications démarrent au prochain cycle, toutes les 60 secondes. Le reste se règle sur la sonde elle-même.',
		'Las comprobaciones empiezan en el siguiente ciclo, cada 60 segundos. El resto se ajusta en el propio monitor.',
		'Die Prüfungen starten im nächsten Takt, alle 60 Sekunden. Alles Weitere stellen Sie am Monitor selbst ein.'
	],
	'drawer.nameHelper': [
		'How it appears in your monitor list.',
		'Son nom dans votre liste de sondes.',
		'Cómo aparece en tu lista de monitores.',
		'So erscheint er in Ihrer Monitorliste.'
	],
	'drawer.httpHelper': [
		'A 200 with any body counts as up until you narrow it.',
		'Un 200 avec n’importe quel corps compte comme disponible tant que vous ne précisez rien.',
		'Un 200 con cualquier cuerpo cuenta como activo hasta que lo concretes.',
		'Ein 200 mit beliebigem Inhalt gilt als online, bis Sie es eingrenzen.'
	],
	'drawer.tcpHelper': [
		'Sonde only opens the connection, it sends nothing.',
		'Sonde ouvre seulement la connexion, elle n’envoie rien.',
		'Sonde solo abre la conexión, no envía nada.',
		'Sonde öffnet nur die Verbindung und sendet nichts.'
	],
	'drawer.push.title': ['Push monitor', 'Sonde push', 'Monitor push', 'Push-Monitor'],
	'drawer.push.text': [
		'Sonde issues a heartbeat URL once the monitor exists. Silence past the interval counts as down.',
		'Sonde fournit une URL de signal de vie une fois la sonde créée. Un silence au-delà de l’intervalle compte comme une panne.',
		'Sonde emite una URL de latido cuando el monitor existe. El silencio más allá del intervalo cuenta como caída.',
		'Sonde stellt eine Herzschlag-URL aus, sobald der Monitor existiert. Schweigen über das Intervall hinaus gilt als Ausfall.'
	],
	'drawer.notAdded': ['Not added', 'Non ajoutée', 'No añadido', 'Nicht hinzugefügt'],
	'drawer.createFailed': [
		'Could not create the monitor.',
		'Impossible de créer la sonde.',
		'No se pudo crear el monitor.',
		'Der Monitor konnte nicht erstellt werden.'
	],
	'drawer.adding': ['Adding…', 'Ajout…', 'Añadiendo…', 'Wird hinzugefügt…'],
	'drawer.add': ['Add monitor', 'Ajouter la sonde', 'Añadir monitor', 'Monitor hinzufügen'],

	'history.title': ['History', 'Historique', 'Historial', 'Verlauf'],
	'history.description': [
		'Coverage, latency and failures over the chosen window.',
		'Couverture, latence et échecs sur la période choisie.',
		'Cobertura, latencia y fallos en el periodo elegido.',
		'Abdeckung, Latenz und Ausfälle im gewählten Zeitraum.'
	],
	'history.window': ['History window', 'Période de l’historique', 'Periodo del historial', 'Verlaufszeitraum'],
	'history.uptimeHistory': [
		'Uptime history',
		'Historique de disponibilité',
		'Historial de disponibilidad',
		'Verfügbarkeitsverlauf'
	],
	'history.response': ['Response time', 'Temps de réponse', 'Tiempo de respuesta', 'Antwortzeit'],
	'history.latencyHour': [
		'Average latency per hour',
		'Latence moyenne par heure',
		'Latencia media por hora',
		'Durchschnittliche Latenz pro Stunde'
	],
	'history.latencyDay': [
		'Average latency per day',
		'Latence moyenne par jour',
		'Latencia media por día',
		'Durchschnittliche Latenz pro Tag'
	],
	'history.failedChecks': ['Failed checks', 'Vérifications échouées', 'Comprobaciones fallidas', 'Fehlgeschlagene Prüfungen'],
	'history.failedHour': [
		'Probes that did not pass, per hour',
		'Mesures en échec, par heure',
		'Mediciones fallidas, por hora',
		'Nicht bestandene Prüfungen, pro Stunde'
	],
	'history.failedDay': [
		'Probes that did not pass, per day',
		'Mesures en échec, par jour',
		'Mediciones fallidas, por día',
		'Nicht bestandene Prüfungen, pro Tag'
	],
	'history.avgLatency': ['Average latency', 'Latence moyenne', 'Latencia media', 'Durchschnittliche Latenz'],
	'history.noChecks': [
		'No checks in this window',
		'Aucune vérification sur cette période',
		'Sin comprobaciones en este periodo',
		'Keine Prüfungen in diesem Zeitraum'
	],
	'history.loadFailed': [
		'Failed to load the history.',
		'Échec du chargement de l’historique.',
		'Error al cargar el historial.',
		'Der Verlauf konnte nicht geladen werden.'
	],

	'uptimeBar.notProbed': ['{when}: not probed', '{when} : non mesuré', '{when}: sin medir', '{when}: nicht gemessen'],
	'uptimeBar.notProbedShort': ['not probed', 'non mesuré', 'sin medir', 'nicht gemessen'],
	'uptimeBar.describe': [
		'{when}: {uptime} of {total} checks, {latency} average',
		'{when} : {uptime} sur {total} vérifications, {latency} en moyenne',
		'{when}: {uptime} de {total} comprobaciones, {latency} de media',
		'{when}: {uptime} bei {total} Prüfungen, durchschnittlich {latency}'
	],
	'uptimeBar.covered': ['{percent} covered', '{percent} couvert', '{percent} cubierto', '{percent} abgedeckt'],
	'uptimeBar.down': ['{duration} down', '{duration} hors ligne', '{duration} caído', '{duration} offline'],
	'uptimeBar.noDowntime': ['no downtime', 'aucune interruption', 'sin caídas', 'keine Ausfallzeit'],
	'uptimeBar.bucket': ['Bucket', 'Intervalle', 'Intervalo', 'Zeitraum'],
	'uptimeBar.uptime': ['Uptime', 'Disponibilité', 'Disponibilidad', 'Verfügbarkeit'],
	'uptimeBar.checks': ['Checks', 'Vérifications', 'Comprobaciones', 'Prüfungen']
} satisfies Record<string, Row>;
