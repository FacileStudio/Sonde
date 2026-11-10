import type { Row } from '../row';

/** The maintenance page: ranked upgrades, version drift and the repositories behind them. */
export const maintenance = {
	'nav.maintenance': ['Maintenance', 'Maintenance', 'Mantenimiento', 'Wartung'],
	'maintenance.description': [
		'What to fix today across every repository, ranked by what is deployed and exploited.',
		'Ce qu’il faut corriger aujourd’hui sur tous les dépôts, classé selon ce qui est déployé et exploité.',
		'Qué corregir hoy en todos los repositorios, ordenado por lo desplegado y lo explotado.',
		'Was heute in allen Repositories zu beheben ist, sortiert nach Bereitstellung und Ausnutzung.'
	],
	'maintenance.loadError': [
		'Could not load maintenance data',
		'Impossible de charger la maintenance',
		'No se pudieron cargar los datos de mantenimiento',
		'Wartungsdaten konnten nicht geladen werden'
	],
	'maintenance.forbidden': [
		'Maintenance data is limited to allow-listed administrators.',
		'La maintenance est réservée aux administrateurs autorisés.',
		'El mantenimiento está limitado a administradores autorizados.',
		'Wartungsdaten sind freigegebenen Administratoren vorbehalten.'
	],
	'maintenance.empty.title': ['No report yet', 'Aucun rapport', 'Aún no hay informes', 'Noch kein Bericht'],
	'maintenance.empty.description': [
		'The nightly scan has not delivered anything. Setup is in docs/FLEET.md.',
		'Le scan nocturne n’a encore rien livré. La mise en place est décrite dans docs/FLEET.md.',
		'El análisis nocturno aún no ha entregado nada. La configuración está en docs/FLEET.md.',
		'Der nächtliche Scan hat noch nichts geliefert. Die Einrichtung steht in docs/FLEET.md.'
	],
	'maintenance.stale.title': ['The last scan is old', 'Le dernier scan est ancien', 'El último análisis es antiguo', 'Der letzte Scan ist alt'],
	'maintenance.stale.body': [
		'The newest report finished {when}. Anything published since is not here yet.',
		'Le rapport le plus récent a fini {when}. Ce qui a été publié depuis n’y figure pas.',
		'El informe más reciente terminó {when}. Lo publicado después aún no aparece.',
		'Der neueste Bericht endete {when}. Was seitdem veröffentlicht wurde, fehlt noch.'
	],
	'maintenance.incomplete.title': ['The last scan was incomplete', 'Le dernier scan était incomplet', 'El último análisis fue incompleto', 'Der letzte Scan war unvollständig'],
	'maintenance.incomplete.body': [
		'{scanned} of {expected} repositories were scanned. The rest keep their previous findings.',
		'{scanned} dépôts sur {expected} ont été analysés. Les autres gardent leurs résultats précédents.',
		'Se analizaron {scanned} de {expected} repositorios. El resto conserva sus hallazgos anteriores.',
		'{scanned} von {expected} Repositories wurden gescannt. Die übrigen behalten ihre bisherigen Befunde.'
	],
	'maintenance.stat.act': ['Act now', 'À traiter', 'Actuar ya', 'Jetzt handeln'],
	'maintenance.stat.actHint': ['Exploited or critical in production', 'Exploité ou critique en production', 'Explotado o crítico en producción', 'Ausgenutzt oder kritisch in Produktion'],
	'maintenance.stat.schedule': ['Schedule', 'À planifier', 'Planificar', 'Einplanen'],
	'maintenance.stat.scheduleHint': ['Serious, not urgent', 'Sérieux, pas urgent', 'Serio, no urgente', 'Ernst, nicht dringend'],
	'maintenance.stat.lastRun': ['Last scan', 'Dernier scan', 'Último análisis', 'Letzter Scan'],
	'maintenance.stat.coverage': [
		'{scanned} of {expected} scanned',
		'{scanned} sur {expected} analysés',
		'{scanned} de {expected} analizados',
		'{scanned} von {expected} gescannt'
	],
	'maintenance.stat.open': ['Open findings', 'Failles ouvertes', 'Hallazgos abiertos', 'Offene Befunde'],
	'maintenance.stat.openHint': [
		{ one: '{count} upgrade clears them', other: '{count} upgrades clear them' },
		{ one: '{count} mise à jour les corrige', other: '{count} mises à jour les corrigent' },
		{ one: '{count} actualización los resuelve', other: '{count} actualizaciones los resuelven' },
		{ one: '{count} Update behebt sie', other: '{count} Updates beheben sie' }
	],
	'maintenance.tab.actions': ['Actions', 'Actions', 'Acciones', 'Aktionen'],
	'maintenance.tab.drift': ['Drift', 'Écarts', 'Desviación', 'Abweichung'],
	'maintenance.tab.projects': ['Repositories', 'Dépôts', 'Repositorios', 'Repositories'],
	'maintenance.view': ['Maintenance view', 'Vue maintenance', 'Vista de mantenimiento', 'Wartungsansicht'],
	'maintenance.level.act': ['Act', 'Agir', 'Actuar', 'Handeln'],
	'maintenance.level.schedule': ['Schedule', 'Planifier', 'Planificar', 'Einplanen'],
	'maintenance.level.track': ['Track', 'Suivre', 'Seguir', 'Beobachten'],
	'maintenance.levels': ['Action level', 'Niveau d’action', 'Nivel de acción', 'Handlungsstufe'],
	'maintenance.col.level': ['Level', 'Niveau', 'Nivel', 'Stufe'],
	'maintenance.col.upgrade': ['Upgrade', 'Mise à jour', 'Actualización', 'Update'],
	'maintenance.col.projects': ['Repositories', 'Dépôts', 'Repositorios', 'Repositories'],
	'maintenance.col.findings': ['Findings', 'Failles', 'Hallazgos', 'Befunde'],
	'maintenance.col.worst': ['Worst', 'Pire', 'Peor', 'Schwerste'],
	'maintenance.col.exploit': ['Exploit odds', 'Risque d’exploit', 'Prob. de explotación', 'Ausnutzungsrisiko'],
	'maintenance.noFix': ['no fix yet', 'pas de correctif', 'sin corrección', 'noch kein Fix'],
	'maintenance.kev': ['Exploited', 'Exploitée', 'Explotada', 'Ausgenutzt'],
	'maintenance.more': [
		{ one: '+{count} more', other: '+{count} more' },
		{ one: '+{count} autre', other: '+{count} autres' },
		{ one: '+{count} más', other: '+{count} más' },
		{ one: '+{count} weiteres', other: '+{count} weitere' }
	],
	'maintenance.actions.empty.title': ['Nothing at this level', 'Rien à ce niveau', 'Nada en este nivel', 'Nichts auf dieser Stufe'],
	'maintenance.actions.empty.description': [
		'No open finding ranks here. Classify repositories to sharpen the order.',
		'Aucune faille ouverte à ce niveau. Classez les dépôts pour affiner l’ordre.',
		'Ningún hallazgo abierto en este nivel. Clasifica los repositorios para afinar el orden.',
		'Kein offener Befund auf dieser Stufe. Ordne Repositories ein, um die Reihenfolge zu schärfen.'
	],
	'maintenance.drift.latest': ['Latest {version}', 'Dernière {version}', 'Última {version}', 'Neueste {version}'],
	'maintenance.drift.current': [
		{ one: '{count} on the latest', other: '{count} on the latest' },
		{ one: '{count} à jour', other: '{count} à jour' },
		{ one: '{count} en la última', other: '{count} en la última' },
		{ one: '{count} auf dem neuesten Stand', other: '{count} auf dem neuesten Stand' }
	],
	'maintenance.drift.allCurrent': [
		'Every repository is on the latest version.',
		'Tous les dépôts sont à jour.',
		'Todos los repositorios están en la última versión.',
		'Alle Repositories sind auf dem neuesten Stand.'
	],
	'maintenance.drift.held': ['Held', 'Retenu', 'Retenido', 'Gehalten'],
	'maintenance.drift.heldUntil': ['until {date}: {reason}', 'jusqu’au {date} : {reason}', 'hasta {date}: {reason}', 'bis {date}: {reason}'],
	'maintenance.drift.heldUntilBare': ['until {date}', 'jusqu’au {date}', 'hasta {date}', 'bis {date}'],
	'maintenance.drift.empty': [
		'No tracked package has been reported yet.',
		'Aucun paquet suivi n’a encore été signalé.',
		'Aún no se ha informado de ningún paquete seguido.',
		'Noch kein beobachtetes Paket gemeldet.'
	],
	'maintenance.col.repository': ['Repository', 'Dépôt', 'Repositorio', 'Repository'],
	'maintenance.col.tier': ['Role', 'Rôle', 'Rol', 'Rolle'],
	'maintenance.col.coverage': ['Scan', 'Scan', 'Análisis', 'Scan'],
	'maintenance.col.ci': ['CI', 'CI', 'CI', 'CI'],
	'maintenance.col.filet': ['Filet', 'Filet', 'Filet', 'Filet'],
	'maintenance.col.open': ['Open', 'Ouvertes', 'Abiertos', 'Offen'],
	'maintenance.covered': ['Scanned', 'Analysé', 'Analizado', 'Gescannt'],
	'maintenance.filetMissing': ['not in CI', 'absent de la CI', 'fuera de CI', 'nicht in CI'],
	'maintenance.filetWired': ['wired', 'branché', 'conectado', 'eingebunden'],
	'maintenance.notScanned': ['Not scanned', 'Non analysé', 'Sin analizar', 'Nicht gescannt'],
	'maintenance.severity.critical': ['Critical', 'Critique', 'Crítica', 'Kritisch'],
	'maintenance.severity.high': ['High', 'Élevée', 'Alta', 'Hoch'],
	'maintenance.severity.medium': ['Medium', 'Moyenne', 'Media', 'Mittel'],
	'maintenance.severity.low': ['Low', 'Faible', 'Baja', 'Niedrig'],
	'maintenance.severity.unknown': ['Unknown', 'Inconnue', 'Desconocida', 'Unbekannt'],
	'maintenance.ci.success': ['passing', 'réussie', 'correcta', 'erfolgreich'],
	'maintenance.ci.failure': ['failing', 'en échec', 'fallida', 'fehlgeschlagen'],
	'maintenance.ci.cancelled': ['cancelled', 'annulée', 'cancelada', 'abgebrochen'],
	'maintenance.ci.timed_out': ['timed out', 'expirée', 'agotó el tiempo', 'Zeitüberschreitung'],
	'maintenance.ci.action_required': ['action required', 'action requise', 'acción requerida', 'Aktion nötig'],
	'maintenance.ci.startup_failure': ['did not start', 'n’a pas démarré', 'no arrancó', 'nicht gestartet'],
	'maintenance.ci.skipped': ['skipped', 'ignorée', 'omitida', 'übersprungen'],
	'maintenance.ci.neutral': ['neutral', 'neutre', 'neutra', 'neutral'],
	'maintenance.ci.stale': ['stale', 'périmée', 'obsoleta', 'veraltet'],
	'maintenance.ciUnknown': ['no runs', 'aucune exécution', 'sin ejecuciones', 'keine Läufe'],
	'maintenance.tierLabel': ['Role of {name}', 'Rôle de {name}', 'Rol de {name}', 'Rolle von {name}'],
	'maintenance.tierFailed': ['Could not save the role', 'Impossible d’enregistrer le rôle', 'No se pudo guardar el rol', 'Rolle konnte nicht gespeichert werden'],
	'maintenance.tier.unclassified': ['Unclassified', 'Non classé', 'Sin clasificar', 'Nicht eingeordnet'],
	'maintenance.tier.deployed': ['Deployed', 'Déployé', 'Desplegado', 'Bereitgestellt'],
	'maintenance.tier.library': ['Library', 'Bibliothèque', 'Biblioteca', 'Bibliothek'],
	'maintenance.tier.cli': ['CLI', 'CLI', 'CLI', 'CLI'],
	'maintenance.tier.client': ['Client site', 'Site client', 'Sitio de cliente', 'Kundenseite'],
	'maintenance.tier.experiment': ['Experiment', 'Expérience', 'Experimento', 'Experiment'],
	'maintenance.tier.archive': ['Archive', 'Archive', 'Archivo', 'Archiv']
} satisfies Record<string, Row>;
