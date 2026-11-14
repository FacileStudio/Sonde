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
		'HTTP, TCP and cron heartbeat monitors, public status pages, and an alert as soon as a service goes down.',
		'Sondes HTTP, TCP et signaux cron, pages d’état publiques, et une alerte dès qu’un service tombe.',
		'Monitores HTTP, TCP y latidos cron, páginas de estado públicas y una alerta en cuanto un servicio cae.',
		'HTTP-, TCP- und Cron-Herzschlag-Monitore, öffentliche Statusseiten und eine Warnung, sobald ein Dienst ausfällt.'
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
		'Sonde checks your sites, ports and cron jobs, and tells your clients when something breaks.',
		'Sonde surveille vos sites, ports et tâches cron, et prévient vos clients quand ça casse.',
		'Sonde vigila tus sitios, puertos y tareas cron, y avisa a tus clientes cuando algo falla.',
		'Sonde prüft Ihre Websites, Ports und Cron-Jobs und informiert Ihre Kunden, wenn etwas ausfällt.'
	],
	'landing.feature.probes.title': [
		'HTTP, TCP and push monitors',
		'Sondes HTTP, TCP et push',
		'Monitores HTTP, TCP y push',
		'HTTP-, TCP- und Push-Monitore'
	],
	'landing.feature.probes.text': [
		'Check a URL, a port or a cron heartbeat. A job that goes quiet opens an incident.',
		'Testez une URL, un port ou le signal d’une tâche cron. Une tâche muette ouvre un incident.',
		'Comprueba una URL, un puerto o el latido de una tarea cron. Una tarea callada abre un incidente.',
		'Prüfen Sie eine URL, einen Port oder den Herzschlag eines Cron-Jobs. Ein stiller Job eröffnet einen Vorfall.'
	],
	'landing.feature.uptime.title': [
		'Uptime from 24 hours to 90 days',
		'Disponibilité de 24 h à 90 jours',
		'Disponibilidad de 24 h a 90 días',
		'Verfügbarkeit von 24 Stunden bis 90 Tagen'
	],
	'landing.feature.uptime.text': [
		'Hourly and daily rollups answer at once, and old checks get pruned.',
		'Des agrégats horaires et journaliers répondent tout de suite, et les vieilles mesures sont purgées.',
		'Los agregados por hora y por día responden al instante, y las mediciones antiguas se purgan.',
		'Stündliche und tägliche Aggregate antworten sofort, alte Messungen werden gelöscht.'
	],
	'landing.feature.status.title': [
		'Public status pages',
		'Pages d’état publiques',
		'Páginas de estado públicas',
		'Öffentliche Statusseiten'
	],
	'landing.feature.status.text': [
		'One page per project, so clients see the state without asking you.',
		'Une page par projet : vos clients voient l’état sans vous écrire.',
		'Una página por proyecto: tus clientes ven el estado sin preguntarte.',
		'Eine Seite pro Projekt, damit Kunden den Zustand sehen, ohne nachzufragen.'
	],
	'landing.feature.alerts.title': [
		'Alerts through Antenne',
		'Alertes via Antenne',
		'Alertas con Antenne',
		'Alarme über Antenne'
	],
	'landing.feature.alerts.text': [
		'Incidents open and close on their own, and Antenne sends the alert.',
		'Les incidents s’ouvrent et se ferment seuls, et Antenne envoie l’alerte.',
		'Los incidentes se abren y se cierran solos, y Antenne envía la alerta.',
		'Vorfälle öffnen und schließen sich selbst, Antenne verschickt die Warnung.'
	],
	'landing.nav.download': ['CLI', 'CLI', 'CLI', 'CLI'],
	'landing.cta.title': [
		'Watch your first service',
		'Surveillez votre premier service',
		'Vigila tu primer servicio',
		'Überwachen Sie Ihren ersten Dienst'
	],
	'landing.cta.text': [
		'Sign in with your Facile account and add a monitor.',
		'Connectez-vous avec votre compte Facile et ajoutez une sonde.',
		'Inicia sesión con tu cuenta de Facile y añade un monitor.',
		'Melden Sie sich mit Ihrem Facile-Konto an und legen Sie einen Monitor an.'
	],
	'landing.demo.label': [
		'Four monitors turn green, a cron job sends a heartbeat, and one API goes down and recovers.',
		'Quatre sondes passent au vert, une tâche cron envoie un signal, puis une API tombe et revient.',
		'Cuatro monitores se ponen en verde, una tarea cron envía un latido y una API cae y se recupera.',
		'Vier Monitore werden grün, ein Cron-Job sendet einen Herzschlag, eine API fällt aus und erholt sich.'
	],
	'landing.demo.summary': ['{up} of {total} up', '{up} sur {total} en ligne', '{up} de {total} en línea', '{up} von {total} online'],
	'landing.demo.timeout': ['timeout', 'expiré', 'agotado', 'Timeout'],
	'downloads.title': ['Downloads', 'Téléchargements', 'Descargas', 'Downloads'],
	'downloads.heading': ['Install the CLI', 'Installer la CLI', 'Instala la CLI', 'Die CLI installieren'],
	'downloads.intro': [
		'Send heartbeats from cron and read your monitors from the shell.',
		'Envoyez des signaux depuis cron et consultez vos sondes dans le terminal.',
		'Envía latidos desde cron y consulta tus monitores en la terminal.',
		'Senden Sie Herzschläge aus Cron und lesen Sie Ihre Monitore im Terminal.'
	],
	'downloads.cta': ['Install the CLI', 'Installer la CLI', 'Instalar la CLI', 'CLI installieren'],
	'downloads.install.script': ['Install script', 'Script d’installation', 'Script de instalación', 'Installationsskript'],
	'downloads.install.facile': ['With facile', 'Avec facile', 'Con facile', 'Mit facile'],
	'downloads.install.brew': ['Homebrew', 'Homebrew', 'Homebrew', 'Homebrew'],
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
