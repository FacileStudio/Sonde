import type { Row } from '../row';

/** Strings shared by the whole app: buttons, navigation, the sign-in page and relative times. */
export const shell = {
	'common.cancel': ['Cancel', 'Annuler', 'Cancelar', 'Abbrechen'],
	'common.refresh': ['Refresh', 'Actualiser', 'Actualizar', 'Aktualisieren'],
	'common.open': ['Open', 'Ouvrir', 'Abrir', 'Öffnen'],
	'common.view': ['View', 'Voir', 'Ver', 'Ansehen'],
	'common.actions': ['Actions', 'Actions', 'Acciones', 'Aktionen'],
	'common.name': ['Name', 'Nom', 'Nombre', 'Name'],
	'common.type': ['Type', 'Type', 'Tipo', 'Typ'],
	'common.slug': ['Slug', 'Identifiant', 'Identificador', 'Kürzel'],
	'common.never': ['never', 'jamais', 'nunca', 'nie'],
	'common.justNow': ['just now', 'à l’instant', 'ahora mismo', 'gerade eben'],
	'common.enabled': ['Enabled', 'Activé', 'Activado', 'Aktiviert'],
	'common.disabled': ['Disabled', 'Désactivé', 'Desactivado', 'Deaktiviert'],
	'common.saveChanges': ['Save changes', 'Enregistrer les modifications', 'Guardar cambios', 'Änderungen speichern'],
	'common.saving': ['Saving…', 'Enregistrement…', 'Guardando…', 'Wird gespeichert…'],
	'nav.monitors': ['Monitors', 'Sondes', 'Monitores', 'Monitore'],
	'nav.incidents': ['Incidents', 'Incidents', 'Incidentes', 'Vorfälle'],
	'nav.settings': ['Settings', 'Réglages', 'Ajustes', 'Einstellungen'],
	'nav.statusPages': ['Status pages', 'Pages d’état', 'Páginas de estado', 'Statusseiten'],
	'shell.unreachable.title': [
		'Sonde is not answering',
		'Sonde ne répond pas',
		'Sonde no responde',
		'Sonde antwortet nicht'
	],
	'shell.unreachable.description': [
		'The server did not return your account. It may be restarting, or the connection dropped. You are still signed in.',
		'Le serveur n’a pas renvoyé votre compte. Il redémarre peut-être, ou la connexion a été coupée. Vous êtes toujours connecté.',
		'El servidor no devolvió tu cuenta. Puede que se esté reiniciando o que se haya cortado la conexión. Sigues con la sesión iniciada.',
		'Der Server hat Ihr Konto nicht geliefert. Möglicherweise startet er neu oder die Verbindung ist abgebrochen. Sie sind weiterhin angemeldet.'
	],
	'shell.tryAgain': ['Try again', 'Réessayer', 'Reintentar', 'Erneut versuchen'],
	'login.title': ['Sign in to Sonde', 'Connexion à Sonde', 'Iniciar sesión en Sonde', 'Anmelden bei Sonde'],
	'login.tagline': [
		'Is it up? Know in a minute.',
		'Est-ce en ligne ? Réponse en une minute.',
		'¿Está en línea? Respuesta en un minuto.',
		'Ist es online? Antwort in einer Minute.'
	],
	'login.failed': [
		'Sign-in failed. Try again.',
		'La connexion a échoué. Réessayez.',
		'No se pudo iniciar sesión. Inténtalo de nuevo.',
		'Die Anmeldung ist fehlgeschlagen. Bitte erneut versuchen.'
	]
} satisfies Record<string, Row>;
