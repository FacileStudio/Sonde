import type { Row } from '../row';

/** Settings: profile, appearance and the advanced tab. */
export const settings = {
	'settings.description': [
		'Your account, this browser, and this instance.',
		'Votre compte, ce navigateur et cette instance.',
		'Tu cuenta, este navegador y esta instancia.',
		'Ihr Konto, dieser Browser und diese Instanz.'
	],
	'settings.section.profile': ['Profile', 'Profil', 'Perfil', 'Profil'],
	'settings.section.preferences': ['Preferences', 'Préférences', 'Preferencias', 'Voreinstellungen'],
	'settings.section.bus': ['Bus', 'Bus', 'Bus', 'Bus'],
	'settings.section.notifications': ['Notifications', 'Notifications', 'Notificaciones', 'Benachrichtigungen'],
	'settings.section.advanced': ['Advanced', 'Avancé', 'Avanzado', 'Erweitert'],
	'settings.memberSince': ['Member since', 'Membre depuis', 'Miembro desde', 'Mitglied seit'],
	'settings.role.admin': ['Administrator', 'Administrateur', 'Administrador', 'Administrator'],
	'settings.role.member': ['Member', 'Membre', 'Miembro', 'Mitglied'],
	'settings.clear.title': [
		'Clear local data?',
		'Effacer les données locales ?',
		'¿Borrar los datos locales?',
		'Lokale Daten löschen?'
	],
	'settings.clear.description': [
		'The theme preference stored in this browser is deleted and you are signed out. There is no undo. Your monitors, incidents and status pages stay on the server.',
		'La préférence de thème enregistrée dans ce navigateur est supprimée et vous êtes déconnecté. Il n’y a pas d’annulation. Vos sondes, incidents et pages d’état restent sur le serveur.',
		'Se elimina la preferencia de tema guardada en este navegador y se cierra tu sesión. No se puede deshacer. Tus monitores, incidentes y páginas de estado permanecen en el servidor.',
		'Die in diesem Browser gespeicherte Designauswahl wird gelöscht und Sie werden abgemeldet. Das lässt sich nicht rückgängig machen. Ihre Monitore, Vorfälle und Statusseiten bleiben auf dem Server.'
	],
	'settings.clear.confirm': [
		'Clear and sign out',
		'Effacer et se déconnecter',
		'Borrar y cerrar sesión',
		'Löschen und abmelden'
	],

	'advanced.export.title': [
		'Configuration export',
		'Export de la configuration',
		'Exportación de la configuración',
		'Konfigurationsexport'
	],
	'advanced.export.description': [
		'Every monitor and status page as JSON, with no secrets in it. Keep it in git.',
		'Chaque sonde et chaque page d’état en JSON, sans aucun secret. À garder dans git.',
		'Cada monitor y página de estado en JSON, sin secretos. Guárdalo en git.',
		'Jeder Monitor und jede Statusseite als JSON, ohne Geheimnisse. Bewahren Sie es in Git auf.'
	],
	'advanced.export.label': [
		'Download sonde-config.json',
		'Télécharger sonde-config.json',
		'Descargar sonde-config.json',
		'sonde-config.json herunterladen'
	],
	'advanced.export.adminHint': [
		'A snapshot of what this instance is watching, taken now.',
		'Un instantané de ce que cette instance surveille, pris maintenant.',
		'Una instantánea de lo que vigila esta instancia, tomada ahora.',
		'Eine Momentaufnahme dessen, was diese Instanz überwacht, jetzt erstellt.'
	],
	'advanced.export.memberHint': [
		'Administrators only. Ask whoever runs this instance.',
		'Réservé aux administrateurs. Demandez à la personne qui gère cette instance.',
		'Solo administradores. Pregunta a quien gestiona esta instancia.',
		'Nur für Administratoren. Fragen Sie die Person, die diese Instanz betreibt.'
	],
	'advanced.export.button': ['Export', 'Exporter', 'Exportar', 'Exportieren'],
	'advanced.export.failed': ['Export failed.', 'L’export a échoué.', 'La exportación falló.', 'Export fehlgeschlagen.'],
	'advanced.instance.title': ['Instance', 'Instance', 'Instancia', 'Instanz'],
	'advanced.instance.description': [
		'What this browser is talking to.',
		'Ce avec quoi ce navigateur communique.',
		'Con qué se comunica este navegador.',
		'Womit dieser Browser kommuniziert.'
	],
	'advanced.instance.url': ['Instance URL', 'URL de l’instance', 'URL de la instancia', 'Instanz-URL'],
	'advanced.api.base': ['API base path', 'Chemin de base de l’API', 'Ruta base de la API', 'API-Basispfad'],
	'advanced.api.baseHint': [
		'Every request is scoped under it.',
		'Toutes les requêtes passent sous ce chemin.',
		'Todas las peticiones cuelgan de ella.',
		'Jede Anfrage liegt darunter.'
	],
	'advanced.api.reference': ['API reference', 'Référence de l’API', 'Referencia de la API', 'API-Referenz'],
	'advanced.api.referenceHint': [
		'Every endpoint this instance serves.',
		'Chaque point d’accès que sert cette instance.',
		'Cada endpoint que sirve esta instancia.',
		'Jeder Endpunkt, den diese Instanz bereitstellt.'
	],
	'advanced.account.title': ['Account', 'Compte', 'Cuenta', 'Konto'],
	'advanced.account.description': [
		'Facts an operator may ask you for.',
		'Des informations qu’un opérateur peut vous demander.',
		'Datos que un operador puede pedirte.',
		'Angaben, nach denen ein Betreiber fragen kann.'
	],
	'advanced.account.userId': ['User ID', 'Identifiant utilisateur', 'ID de usuario', 'Benutzer-ID'],
	'advanced.account.createdHint': [
		'When this account was created.',
		'Date de création de ce compte.',
		'Cuándo se creó esta cuenta.',
		'Wann dieses Konto erstellt wurde.'
	],
	'advanced.danger.title': ['Danger zone', 'Zone dangereuse', 'Zona de peligro', 'Gefahrenbereich'],
	'advanced.danger.description': [
		'Irreversible from this browser. Your monitors, incidents and status pages are untouched.',
		'Irréversible depuis ce navigateur. Vos sondes, incidents et pages d’état ne sont pas touchés.',
		'Irreversible desde este navegador. Tus monitores, incidentes y páginas de estado no se tocan.',
		'Von diesem Browser aus nicht umkehrbar. Ihre Monitore, Vorfälle und Statusseiten bleiben unberührt.'
	],
	'advanced.danger.clear': ['Clear local data', 'Effacer les données locales', 'Borrar datos locales', 'Lokale Daten löschen'],
} satisfies Record<string, Row>;
