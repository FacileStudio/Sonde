import type { Row } from '../row';

/** Settings: profile, appearance and the advanced tab. */
export const settings = {
	'settings.description': [
		'Your account, this browser, and this instance.',
		'Votre compte, ce navigateur et cette instance.',
		'Tu cuenta, este navegador y esta instancia.',
		'Ihr Konto, dieser Browser und diese Instanz.'
	],
	'settings.sections': [
		'Settings sections',
		'Sections des réglages',
		'Secciones de ajustes',
		'Einstellungsbereiche'
	],
	'settings.section.profile': ['Profile', 'Profil', 'Perfil', 'Profil'],
	'settings.section.appearance': ['Appearance', 'Apparence', 'Apariencia', 'Darstellung'],
	'settings.section.bus': ['Bus', 'Bus', 'Bus', 'Bus'],
	'settings.section.notifications': ['Notifications', 'Notifications', 'Notificaciones', 'Benachrichtigungen'],
	'settings.section.advanced': ['Advanced', 'Avancé', 'Avanzado', 'Erweitert'],
	'settings.memberSince': ['Member since', 'Membre depuis', 'Miembro desde', 'Mitglied seit'],
	'settings.role': ['Role', 'Rôle', 'Rol', 'Rolle'],
	'settings.role.admin': ['Administrator', 'Administrateur', 'Administrador', 'Administrator'],
	'settings.role.member': ['Member', 'Membre', 'Miembro', 'Mitglied'],
	'settings.identity.title': ['Identity', 'Identité', 'Identidad', 'Identität'],
	'settings.identity.description': [
		'How Sonde names you across the app.',
		'Comment Sonde vous nomme dans l’application.',
		'Cómo te nombra Sonde en la aplicación.',
		'Wie Sonde Sie in der Anwendung nennt.'
	],
	'settings.displayName': ['Display name', 'Nom affiché', 'Nombre visible', 'Anzeigename'],
	'settings.namePlaceholder': ['Your name', 'Votre nom', 'Tu nombre', 'Ihr Name'],
	'settings.email': ['Email', 'E-mail', 'Correo electrónico', 'E-Mail'],
	'settings.emailOwned': [
		'Owned by your Facile account at sso.facile.studio. Change it there and it lands here at your next sign-in.',
		'Géré par votre compte Facile sur sso.facile.studio. Modifiez-le là-bas : il arrive ici à votre prochaine connexion.',
		'Lo gestiona tu cuenta Facile en sso.facile.studio. Cámbialo allí y llegará aquí en tu próximo inicio de sesión.',
		'Wird von Ihrem Facile-Konto unter sso.facile.studio verwaltet. Ändern Sie sie dort, dann erscheint sie bei der nächsten Anmeldung hier.'
	],
	'settings.saveProfile': ['Save profile', 'Enregistrer le profil', 'Guardar perfil', 'Profil speichern'],
	'settings.nameRequired': [
		'Enter a display name to save.',
		'Saisissez un nom affiché pour enregistrer.',
		'Introduce un nombre visible para guardar.',
		'Geben Sie zum Speichern einen Anzeigenamen ein.'
	],
	'settings.profileUpdated': [
		'Profile updated.',
		'Profil mis à jour.',
		'Perfil actualizado.',
		'Profil aktualisiert.'
	],
	'settings.profileFailed': [
		'Could not update your profile.',
		'Impossible de mettre à jour votre profil.',
		'No se pudo actualizar tu perfil.',
		'Ihr Profil konnte nicht aktualisiert werden.'
	],
	'settings.session.title': ['Session', 'Session', 'Sesión', 'Sitzung'],
	'settings.session.description': [
		'This browser only. Your monitors keep running either way.',
		'Ce navigateur uniquement. Vos sondes continuent de tourner dans tous les cas.',
		'Solo este navegador. Tus monitores siguen funcionando de todos modos.',
		'Nur dieser Browser. Ihre Monitore laufen in jedem Fall weiter.'
	],
	'settings.logout': ['Log out', 'Se déconnecter', 'Cerrar sesión', 'Abmelden'],
	'settings.logout.description': [
		'You will need to sign in again to reach the dashboard.',
		'Vous devrez vous reconnecter pour accéder au tableau de bord.',
		'Tendrás que volver a iniciar sesión para acceder al panel.',
		'Sie müssen sich erneut anmelden, um das Dashboard zu erreichen.'
	],
	'settings.logout.title': ['Log out of Sonde?', 'Se déconnecter de Sonde ?', '¿Cerrar sesión en Sonde?', 'Von Sonde abmelden?'],
	'settings.logout.confirmText': [
		'Your monitors keep running. You will need to sign in again on this browser.',
		'Vos sondes continuent de tourner. Vous devrez vous reconnecter sur ce navigateur.',
		'Tus monitores siguen funcionando. Tendrás que volver a iniciar sesión en este navegador.',
		'Ihre Monitore laufen weiter. Sie müssen sich in diesem Browser erneut anmelden.'
	],
	'settings.theme.title': ['Theme', 'Thème', 'Tema', 'Design'],
	'settings.theme.description': [
		'Applied to this browser. Every device you sign in from keeps its own choice.',
		'Appliqué à ce navigateur. Chaque appareil garde son propre choix.',
		'Se aplica a este navegador. Cada dispositivo conserva su propia elección.',
		'Gilt für diesen Browser. Jedes Gerät behält seine eigene Auswahl.'
	],
	'settings.theme.scheme': ['Colour scheme', 'Schéma de couleurs', 'Esquema de color', 'Farbschema'],
	'settings.theme.schemeHint': [
		'System follows your OS for as long as the app is open.',
		'« Système » suit votre système d’exploitation tant que l’application est ouverte.',
		'«Sistema» sigue a tu sistema operativo mientras la aplicación esté abierta.',
		'„System“ folgt Ihrem Betriebssystem, solange die Anwendung geöffnet ist.'
	],
	'settings.theme.system': ['System', 'Système', 'Sistema', 'System'],
	'settings.theme.light': ['Light', 'Clair', 'Claro', 'Hell'],
	'settings.theme.dark': ['Dark', 'Sombre', 'Oscuro', 'Dunkel'],
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
	'advanced.danger.clearHint': [
		'Removes the theme preference stored here, then signs you out. To delete your account and its data, delete your Facile account in Registre: Sonde then erases your monitors, webhooks and sessions.',
		'Supprime la préférence de thème enregistrée ici, puis vous déconnecte. Pour supprimer votre compte et ses données, supprimez votre compte Facile dans Registre : Sonde efface alors vos sondes, webhooks et sessions.',
		'Elimina la preferencia de tema guardada aquí y cierra tu sesión. Para eliminar tu cuenta y sus datos, elimina tu cuenta Facile en Registre: Sonde borrará entonces tus monitores, webhooks y sesiones.',
		'Entfernt die hier gespeicherte Designauswahl und meldet Sie ab. Um Ihr Konto und seine Daten zu löschen, löschen Sie Ihr Facile-Konto in Registre: Sonde entfernt dann Ihre Monitore, Webhooks und Sitzungen.'
	]
} satisfies Record<string, Row>;
