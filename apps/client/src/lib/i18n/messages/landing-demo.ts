import type { Row } from '../row';

/** The landing demos, their three tiles and the CLI install labels. */
export const landingDemo = {
	'landing.demo.label': [
		'Four monitors turn green, a cron job sends a heartbeat, and one API goes down and recovers.',
		'Quatre sondes passent au vert, une tâche cron envoie un signal, puis une API tombe et revient.',
		'Cuatro monitores se ponen en verde, una tarea cron envía un latido y una API cae y se recupera.',
		'Vier Monitore werden grün, ein Cronjob sendet einen Herzschlag, eine API fällt aus und erholt sich.'
	],
	'landing.demo.summary': ['{up} of {total} up', '{up} sur {total} en ligne', '{up} de {total} en línea', '{up} von {total} online'],
	'landing.demo.timeout': ['timeout', 'expiré', 'agotado', 'Timeout'],
	'landing.demo.incident': ['Incident open', 'Incident ouvert', 'Incidente abierto', 'Vorfall offen'],
	'landing.demo.failed': ['Check failed {count} of 3', 'Échec {count} sur 3', 'Fallo {count} de 3', 'Fehlschlag {count} von 3'],
	'landing.demo.alert': ['Alert sent to your phone', 'Alerte envoyée sur votre téléphone', 'Alerta enviada a tu móvil', 'Alarm aufs Telefon gesendet'],
	'landing.demo.rowSite': ['Website', 'Site vitrine', 'Web', 'Website'],
	'landing.demo.rowShop': ['Online shop', 'Boutique en ligne', 'Tienda online', 'Onlineshop'],
	'landing.demo.rowBooking': ['Bookings', 'Réservations', 'Reservas', 'Buchungen'],
	'landing.demo.lastSignal': ['Last heartbeat at 03:00', 'Dernier signal à 03:00', 'Último latido a las 03:00', 'Letzter Herzschlag um 03:00'],
	'landing.demo.noSignal': ['No heartbeat for {hours} h', 'Aucun signal depuis {hours} h', 'Sin latido desde hace {hours} h', 'Kein Herzschlag seit {hours} h'],
	'landing.card.down.title': [
		'A site goes down, you know within the minute',
		'Un site tombe, vous le savez dans la minute',
		'Una web cae y lo sabes en un minuto',
		'Eine Website fällt aus, Sie wissen es in einer Minute'
	],
	'landing.card.down.body': [
		'Checked every 20 seconds, three failures in a row open an incident and send the alert.',
		'Avec un test toutes les 20 secondes, trois échecs de suite ouvrent un incident et envoient l’alerte.',
		'Con una prueba cada 20 segundos, tres fallos seguidos abren un incidente y envían la alerta.',
		'Bei einer Prüfung alle 20 Sekunden eröffnen drei Fehlschläge in Folge einen Vorfall und lösen den Alarm aus.'
	],
	'landing.card.status.title': [
		'A public status page for each client',
		'Une page d’état publique pour chaque client',
		'Una página de estado pública para cada cliente',
		'Eine öffentliche Statusseite für jeden Kunden'
	],
	'landing.card.status.body': [
		'Group a client’s services on one page. They see what works without writing to you.',
		'Regroupez les services d’un client sur une page. Il voit ce qui marche sans vous écrire.',
		'Agrupa los servicios de un cliente en una página. Ve lo que funciona sin escribirte.',
		'Bündeln Sie die Dienste eines Kunden auf einer Seite. Er sieht, was läuft, ohne Ihnen zu schreiben.'
	],
	'landing.card.cron.title': [
		'A cron job that stops answering opens an incident',
		'Une tâche cron qui ne répond plus ouvre un incident',
		'Una tarea cron que deja de responder abre un incidente',
		'Ein Cronjob, der nicht mehr antwortet, eröffnet einen Vorfall'
	],
	'landing.card.cron.body': [
		'Your backup sends a heartbeat each run. When it goes quiet, Sonde tells you.',
		'Votre sauvegarde envoie un signal à chaque passage. Quand il se tait, Sonde vous prévient.',
		'Tu copia de seguridad envía un latido en cada pasada. Cuando calla, Sonde te avisa.',
		'Ihr Backup sendet bei jedem Lauf einen Herzschlag. Bleibt er aus, meldet sich Sonde.'
	],
	'downloads.title': ['Downloads', 'Téléchargements', 'Descargas', 'Downloads'],
	'downloads.intro': [
		'Send heartbeats from cron and read your monitors from the shell.',
		'Envoyez des signaux depuis cron et consultez vos sondes dans le terminal.',
		'Envía latidos desde cron y consulta tus monitores en la terminal.',
		'Senden Sie Herzschläge aus Cron und lesen Sie Ihre Monitore im Terminal.'
	],
	'downloads.install.script': ['Install script', 'Script d’installation', 'Script de instalación', 'Installationsskript'],
	'downloads.install.kit': ['With kit', 'Avec kit', 'Con kit', 'Mit kit'],
	'downloads.install.brew': ['Homebrew', 'Homebrew', 'Homebrew', 'Homebrew']
} satisfies Record<string, Row>;
