import type { Row } from '../row';

/** The live panel the maintenance page shows while a fleet scan runs. */
export const maintenanceScan = {
	'maintenance.scan.title': ['Fleet scan', 'Scan de la flotte', 'Análisis de la flota', 'Flotten-Scan'],
	'maintenance.scan.state.queued': [
		'Waiting for a GitHub runner',
		'En attente d’un runner GitHub',
		'Esperando un runner de GitHub',
		'Wartet auf einen GitHub-Runner'
	],
	'maintenance.scan.state.in_progress': ['Running', 'En cours', 'En curso', 'Läuft'],
	'maintenance.scan.state.delivering': [
		'Delivering the report',
		'Envoi du rapport',
		'Entregando el informe',
		'Bericht wird übermittelt'
	],
	'maintenance.scan.phase.listing': [
		'Listing repositories',
		'Liste des dépôts',
		'Listando repositorios',
		'Repositories werden aufgelistet'
	],
	'maintenance.scan.phase.scanning': [
		'Reading repositories',
		'Lecture des dépôts',
		'Leyendo repositorios',
		'Repositories werden gelesen'
	],
	'maintenance.scan.phase.douane': [
		'Checking advisories with douane',
		'Vérification des alertes avec douane',
		'Comprobando avisos con douane',
		'Sicherheitshinweise werden mit douane geprüft'
	],
	'maintenance.scan.phase.delivering': [
		'Storing the report in Sonde',
		'Enregistrement du rapport dans Sonde',
		'Guardando el informe en Sonde',
		'Bericht wird in Sonde gespeichert'
	],
	'maintenance.scan.repos': [
		'{done} of {total} repositories',
		'{done} dépôts sur {total}',
		'{done} de {total} repositorios',
		'{done} von {total} Repositories'
	],
	'maintenance.scan.current': ['Now on {name}', 'En cours : {name}', 'Ahora en {name}', 'Gerade bei {name}'],
	'maintenance.scan.waiting': [
		'Waiting for the runner to report',
		'En attente des nouvelles du runner',
		'Esperando noticias del runner',
		'Wartet auf eine Meldung des Runners'
	],
	'maintenance.scan.elapsed': ['Elapsed {time}', 'Écoulé {time}', 'Transcurrido {time}', 'Vergangen {time}'],
	'maintenance.scan.steps': ['Job steps', 'Étapes du job', 'Pasos del job', 'Job-Schritte'],
	'maintenance.scan.openRun': [
		'Open the run on GitHub',
		'Ouvrir le run sur GitHub',
		'Abrir la ejecución en GitHub',
		'Lauf auf GitHub öffnen'
	],
	'maintenance.scan.failed.title': ['The scan failed', 'Le scan a échoué', 'El análisis falló', 'Der Scan ist fehlgeschlagen'],
	'maintenance.scan.failed.body': [
		'GitHub reports the run as {conclusion}. No new report was stored.',
		'GitHub indique le run comme {conclusion}. Aucun nouveau rapport n’a été enregistré.',
		'GitHub marca la ejecución como {conclusion}. No se guardó ningún informe nuevo.',
		'GitHub meldet den Lauf als {conclusion}. Es wurde kein neuer Bericht gespeichert.'
	],
	'maintenance.scan.finished': [
		'Scan finished: {repos} repos, {fresh} new findings, {resolved} resolved',
		'Scan terminé : {repos} dépôts, {fresh} nouvelles failles, {resolved} résolues',
		'Análisis terminado: {repos} repositorios, {fresh} hallazgos nuevos, {resolved} resueltos',
		'Scan fertig: {repos} Repositories, {fresh} neue Befunde, {resolved} behoben'
	],
	'maintenance.scan.finishedShort': [
		'Scan finished: {repos} repos',
		'Scan terminé : {repos} dépôts',
		'Análisis terminado: {repos} repositorios',
		'Scan fertig: {repos} Repositories'
	],
	'maintenance.scan.noReport': [
		'The run finished but no new report arrived.',
		'Le run est terminé mais aucun nouveau rapport n’est arrivé.',
		'La ejecución terminó pero no llegó ningún informe nuevo.',
		'Der Lauf ist fertig, aber kein neuer Bericht ist eingetroffen.'
	]
} satisfies Record<string, Row>;
