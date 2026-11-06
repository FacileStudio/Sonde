import type { Row } from '../row';

/** The status page admin screen, its drawer and the public status page. */
export const status = {
	'statusPages.description': [
		'Public pages showing which of your projects are up. No account needed to read them.',
		'Des pages publiques qui montrent lesquels de vos projets sont disponibles. Aucun compte n’est nécessaire pour les lire.',
		'Páginas públicas que muestran qué proyectos están activos. No hace falta cuenta para leerlas.',
		'Öffentliche Seiten, die zeigen, welche Ihrer Projekte laufen. Zum Lesen ist kein Konto nötig.'
	],
	'statusPages.new': ['New status page', 'Nouvelle page d’état', 'Nueva página de estado', 'Neue Statusseite'],
	'statusPages.adminOnly.title': [
		'Administrators only',
		'Réservé aux administrateurs',
		'Solo administradores',
		'Nur für Administratoren'
	],
	'statusPages.adminOnly.description': [
		'Status pages are managed by whoever runs this instance. Ask them for access, or for a link to a page that is already published.',
		'Les pages d’état sont gérées par la personne qui administre cette instance. Demandez-lui un accès, ou le lien d’une page déjà publiée.',
		'Las páginas de estado las gestiona quien administra esta instancia. Pídele acceso o el enlace de una página ya publicada.',
		'Statusseiten werden von der Person verwaltet, die diese Instanz betreibt. Bitten Sie um Zugang oder um den Link zu einer bereits veröffentlichten Seite.'
	],
	'statusPages.loadError': [
		'Could not load status pages',
		'Impossible de charger les pages d’état',
		'No se pudieron cargar las páginas de estado',
		'Statusseiten konnten nicht geladen werden'
	],
	'statusPages.loadFailed': [
		'Failed to load status pages.',
		'Échec du chargement des pages d’état.',
		'Error al cargar las páginas de estado.',
		'Statusseiten konnten nicht geladen werden.'
	],
	'statusPages.empty.title': [
		'No public pages yet',
		'Aucune page publique pour l’instant',
		'Aún no hay páginas públicas',
		'Noch keine öffentlichen Seiten'
	],
	'statusPages.empty.description': [
		'Publish a page so clients can see uptime without an account.',
		'Publiez une page pour que vos clients voient la disponibilité sans compte.',
		'Publica una página para que los clientes vean la disponibilidad sin cuenta.',
		'Veröffentlichen Sie eine Seite, damit Kunden die Verfügbarkeit ohne Konto sehen.'
	],
	'statusPages.empty.cta': [
		'Create a status page',
		'Créer une page d’état',
		'Crear una página de estado',
		'Statusseite erstellen'
	],
	'statusPages.col.title': ['Title', 'Titre', 'Título', 'Titel'],
	'statusPages.col.address': ['Address', 'Adresse', 'Dirección', 'Adresse'],
	'statusPages.col.monitors': ['Monitors', 'Sondes', 'Monitores', 'Monitore'],
	'statusPages.editAria': ['Edit {title}', 'Modifier {title}', 'Editar {title}', '{title} bearbeiten'],
	'statusPages.deleteAria': ['Delete {title}', 'Supprimer {title}', 'Eliminar {title}', '{title} löschen'],
	'statusPages.saved': ['{title} saved.', '{title} enregistrée.', '{title} guardada.', '{title} gespeichert.'],
	'statusPages.live': [
		'{title} is live at /status/{slug}.',
		'{title} est en ligne sur /status/{slug}.',
		'{title} está publicada en /status/{slug}.',
		'{title} ist unter /status/{slug} online.'
	],
	'statusPages.takenDown': ['{title} taken down.', '{title} retirée.', '{title} retirada.', '{title} entfernt.'],
	'statusPages.deleteFailed': [
		'Could not delete the status page.',
		'Impossible de supprimer la page d’état.',
		'No se pudo eliminar la página de estado.',
		'Die Statusseite konnte nicht gelöscht werden.'
	],
	'statusPages.delete.title': ['Delete {title}?', 'Supprimer {title} ?', '¿Eliminar {title}?', '{title} löschen?'],
	'statusPages.delete.titleFallback': [
		'Delete status page?',
		'Supprimer la page d’état ?',
		'¿Eliminar la página de estado?',
		'Statusseite löschen?'
	],
	'statusPages.delete.description': [
		'The public address stops resolving immediately. The monitors it lists are untouched.',
		'L’adresse publique cesse de répondre immédiatement. Les sondes listées ne sont pas modifiées.',
		'La dirección pública deja de funcionar de inmediato. Los monitores que lista no se modifican.',
		'Die öffentliche Adresse ist sofort nicht mehr erreichbar. Die aufgeführten Monitore bleiben unberührt.'
	],
	'statusPages.delete.confirm': ['Delete page', 'Supprimer la page', 'Eliminar página', 'Seite löschen'],

	'statusDrawer.edit': ['Edit status page', 'Modifier la page d’état', 'Editar página de estado', 'Statusseite bearbeiten'],
	'statusDrawer.description': [
		'Anyone with the link sees these monitors and their uptime, in this order.',
		'Toute personne disposant du lien voit ces sondes et leur disponibilité, dans cet ordre.',
		'Cualquiera con el enlace ve estos monitores y su disponibilidad, en este orden.',
		'Jeder mit dem Link sieht diese Monitore und ihre Verfügbarkeit, in dieser Reihenfolge.'
	],
	'statusDrawer.titleHelper': [
		'Shown as the heading of the public page.',
		'Affiché comme titre de la page publique.',
		'Se muestra como encabezado de la página pública.',
		'Wird als Überschrift der öffentlichen Seite angezeigt.'
	],
	'statusDrawer.titlePlaceholder': ['Acme status', 'État d’Acme', 'Estado de Acme', 'Acme-Status'],
	'statusDrawer.slugHelper': [
		'The public address is /status/ followed by this.',
		'L’adresse publique est /status/ suivi de cet identifiant.',
		'La dirección pública es /status/ seguido de esto.',
		'Die öffentliche Adresse lautet /status/ gefolgt von diesem Kürzel.'
	],
	'statusDrawer.moveUp': ['Move {name} up', 'Monter {name}', 'Subir {name}', '{name} nach oben'],
	'statusDrawer.moveDown': ['Move {name} down', 'Descendre {name}', 'Bajar {name}', '{name} nach unten'],
	'statusDrawer.removeFrom': [
		'Remove {name} from this page',
		'Retirer {name} de cette page',
		'Quitar {name} de esta página',
		'{name} von dieser Seite entfernen'
	],
	'statusDrawer.noMonitors': [
		'No monitors to publish yet.',
		'Aucune sonde à publier pour l’instant.',
		'Aún no hay monitores que publicar.',
		'Noch keine Monitore zum Veröffentlichen.'
	],
	'statusDrawer.saveFailed': [
		'Could not save the status page.',
		'Impossible d’enregistrer la page d’état.',
		'No se pudo guardar la página de estado.',
		'Die Statusseite konnte nicht gespeichert werden.'
	],
	'statusDrawer.create': ['Create page', 'Créer la page', 'Crear página', 'Seite erstellen'],

	'public.title': ['{title} · Status', '{title} · État', '{title} · Estado', '{title} · Status'],
	'public.titleFallback': ['Status · Sonde', 'État · Sonde', 'Estado · Sonde', 'Status · Sonde'],
	'public.notFound.title': ['Page not found', 'Page introuvable', 'Página no encontrada', 'Seite nicht gefunden'],
	'public.notFound.description': [
		'No public status page lives at this address.',
		'Aucune page d’état publique n’existe à cette adresse.',
		'No hay ninguna página de estado pública en esta dirección.',
		'Unter dieser Adresse gibt es keine öffentliche Statusseite.'
	],
	'public.readAt': [
		'Read at {time}, and again every minute.',
		'Lu à {time}, puis à nouveau chaque minute.',
		'Leído a las {time} y de nuevo cada minuto.',
		'Gelesen um {time} und danach jede Minute erneut.'
	],
	'public.refreshed': [
		'Refreshed every minute.',
		'Actualisé chaque minute.',
		'Se actualiza cada minuto.',
		'Wird jede Minute aktualisiert.'
	],
	'public.empty.title': [
		'Nothing published yet',
		'Rien de publié pour l’instant',
		'Aún no hay nada publicado',
		'Noch nichts veröffentlicht'
	],
	'public.empty.description': [
		'This status page has no monitors on it.',
		'Cette page d’état ne contient aucune sonde.',
		'Esta página de estado no tiene monitores.',
		'Diese Statusseite enthält keine Monitore.'
	],
	'public.verdict.down': [
		'Some systems are down',
		'Certains systèmes sont hors ligne',
		'Algunos sistemas están caídos',
		'Einige Systeme sind ausgefallen'
	],
	'public.verdict.unknown': [
		'Some systems have not reported yet',
		'Certains systèmes ne se sont pas encore manifestés',
		'Algunos sistemas aún no han informado',
		'Einige Systeme haben sich noch nicht gemeldet'
	],
	'public.verdict.ok': [
		'All systems operational',
		'Tous les systèmes fonctionnent',
		'Todos los sistemas operativos',
		'Alle Systeme laufen'
	],
	'public.verdict.downText': [
		'{down} of {total} services are failing their checks. We are on it.',
		'{down} services sur {total} échouent à leurs vérifications. Nous nous en occupons.',
		'{down} de {total} servicios fallan sus comprobaciones. Estamos en ello.',
		'{down} von {total} Diensten bestehen ihre Prüfungen nicht. Wir kümmern uns darum.'
	],
	'public.verdict.okText': [
		'{up} of {total} services are answering normally.',
		'{up} services sur {total} répondent normalement.',
		'{up} de {total} servicios responden con normalidad.',
		'{up} von {total} Diensten antworten normal.'
	],
	'public.stat.operational': ['Operational', 'Opérationnels', 'Operativos', 'Betriebsbereit'],
	'public.stat.openIncidents': ['Open incidents', 'Incidents ouverts', 'Incidentes abiertos', 'Offene Vorfälle'],
	'public.stat.beingWorked': ['Being worked on', 'En cours de traitement', 'En tratamiento', 'In Bearbeitung'],
	'public.stat.average': [
		'Average uptime, 24h',
		'Disponibilité moyenne, 24 h',
		'Disponibilidad media, 24 h',
		'Durchschnittliche Verfügbarkeit, 24 h'
	],
	'public.stat.across': ['Across {count} services', 'Sur {count} services', 'En {count} servicios', 'Über {count} Dienste'],
	'public.byState': ['Services by state', 'Services par état', 'Servicios por estado', 'Dienste nach Status'],
	'public.slice.unknown': ['Unknown', 'Inconnu', 'Desconocido', 'Unbekannt'],
	'public.operational': ['operational', 'opérationnels', 'operativos', 'betriebsbereit'],
	'public.badge24': ['{percent} · 24h', '{percent} · 24 h', '{percent} · 24 h', '{percent} · 24 h'],
	'public.badge7': ['{percent} · 7d', '{percent} · 7 j', '{percent} · 7 d', '{percent} · 7 T'],
	'public.failingChecks': ['Failing checks', 'Vérifications en échec', 'Comprobaciones fallidas', 'Fehlgeschlagene Prüfungen'],
	'public.failingSince': [
		'{cause}, since {when}',
		'{cause}, depuis le {when}',
		'{cause}, desde {when}',
		'{cause}, seit {when}'
	],
	'public.poweredBy': ['Powered by Sonde', 'Propulsé par Sonde', 'Con tecnología de Sonde', 'Bereitgestellt von Sonde']
} satisfies Record<string, Row>;
