import type { Row } from '../row';

/** The public landing page. */
export const landing = {
	'landing.title': [
		'Sonde · Uptime monitoring and status pages',
		'Sonde · Surveillance de disponibilité et pages de statut',
		'Sonde · Monitorización de disponibilidad y páginas de estado',
		'Sonde · Verfügbarkeitsüberwachung und Statusseiten'
	],
	'landing.description': [
		'Heartbeats, multi-region probes and a public status page, with an alert as soon as a service goes down.',
		'Heartbeats, sondes multi-régions et page de statut publique, avec une alerte dès qu’un service tombe.',
		'Heartbeats, sondas multirregión y una página de estado pública, con una alerta en cuanto un servicio cae.',
		'Heartbeats, Prüfungen aus mehreren Regionen und eine öffentliche Statusseite, mit einer Warnung, sobald ein Dienst ausfällt.'
	],
	'landing.nav.features': ['Features', 'Fonctions', 'Funciones', 'Funktionen'],
	'landing.nav.pricing': ['Pricing', 'Tarifs', 'Precios', 'Preise'],
	'landing.hero.line1': ['Is it up?', 'Est-ce en ligne ?', '¿Está en línea?', 'Ist es online?'],
	'landing.hero.line2': [
		'Know in a minute.',
		'Réponse en une minute.',
		'Respuesta en un minuto.',
		'Antwort in einer Minute.'
	],
	'landing.hero.text': [
		'Sonde watches your HTTP endpoints, TCP ports and cron jobs, opens incidents when they stop answering, and publishes status pages your clients can read themselves.',
		'Sonde surveille vos points d’accès HTTP, vos ports TCP et vos tâches cron, ouvre un incident quand ils ne répondent plus et publie des pages d’état que vos clients lisent eux-mêmes.',
		'Sonde vigila tus endpoints HTTP, puertos TCP y tareas cron, abre incidentes cuando dejan de responder y publica páginas de estado que tus clientes pueden leer por sí mismos.',
		'Sonde überwacht Ihre HTTP-Endpunkte, TCP-Ports und Cron-Jobs, eröffnet Vorfälle, wenn sie nicht mehr antworten, und veröffentlicht Statusseiten, die Ihre Kunden selbst lesen können.'
	],
	'landing.feature.probes.title': [
		'HTTP, TCP and push monitors',
		'Sondes HTTP, TCP et push',
		'Monitores HTTP, TCP y push',
		'HTTP-, TCP- und Push-Monitore'
	],
	'landing.feature.probes.text': [
		'Probe URLs for status and keywords, check ports, let cron jobs heartbeat. Silence is an incident.',
		'Testez des URL (statut, mot-clé), vérifiez des ports, laissez vos tâches cron envoyer un signal. Le silence est un incident.',
		'Comprueba URL (estado y palabra clave) y puertos, y deja que las tareas cron envíen su latido. El silencio es un incidente.',
		'Prüfen Sie URLs auf Status und Stichwort, testen Sie Ports und lassen Sie Cron-Jobs einen Herzschlag senden. Schweigen ist ein Vorfall.'
	],
	'landing.feature.uptime.title': [
		'Uptime that adds up',
		'Une disponibilité qui se calcule',
		'Una disponibilidad que cuadra',
		'Verfügbarkeit, die stimmt'
	],
	'landing.feature.uptime.text': [
		'Rollup aggregates answer 24h to 90d uptime instantly, without a database the size of a planet.',
		'Des agrégats donnent la disponibilité de 24 h à 90 jours instantanément, sans base de données grosse comme une planète.',
		'Los agregados responden al instante la disponibilidad de 24 h a 90 días, sin una base de datos del tamaño de un planeta.',
		'Aggregierte Werte liefern die Verfügbarkeit von 24 Stunden bis 90 Tagen sofort, ohne eine Datenbank von Planetengröße.'
	],
	'landing.feature.status.title': [
		'Public status pages',
		'Pages d’état publiques',
		'Páginas de estado públicas',
		'Öffentliche Statusseiten'
	],
	'landing.feature.status.text': [
		'Publish per-project status pages so clients see green without asking you.',
		'Publiez une page d’état par projet : vos clients voient que tout va bien sans vous le demander.',
		'Publica una página de estado por proyecto y que tus clientes vean el verde sin preguntarte.',
		'Veröffentlichen Sie pro Projekt eine Statusseite, damit Ihre Kunden Grün sehen, ohne nachzufragen.'
	],
	'landing.feature.alerts.title': [
		'Alerts through Antenne',
		'Alertes via Antenne',
		'Alertas con Antenne',
		'Alarme über Antenne'
	],
	'landing.feature.alerts.text': [
		'Incidents flow straight into the Facile alert bus. No notification zoo included, on purpose.',
		'Les incidents partent directement dans le bus d’alertes Facile. Pas de zoo de notifications, volontairement.',
		'Los incidentes van directamente al bus de alertas de Facile. Sin zoo de notificaciones, a propósito.',
		'Vorfälle fließen direkt in den Facile-Alarmbus. Ein Zoo aus Benachrichtigungen fehlt absichtlich.'
	],
	'landing.tagline': [
		'Uptime monitoring for your services, cron jobs and status pages.',
		'La surveillance de disponibilité de vos services, tâches cron et pages d’état.',
		'Monitorización de disponibilidad para tus servicios, tareas cron y páginas de estado.',
		'Verfügbarkeitsüberwachung für Ihre Dienste, Cron-Jobs und Statusseiten.'
	],
	'landing.column.product': ['Product', 'Produit', 'Producto', 'Produkt'],
	'landing.column.facile': ['Facile', 'Facile', 'Facile', 'Facile'],
	'landing.link.api': ['API reference', 'Référence de l’API', 'Referencia de la API', 'API-Referenz'],
	'landing.pricing': ['Pricing', 'Tarifs', 'Precios', 'Preise']
} satisfies Record<string, Row>;
