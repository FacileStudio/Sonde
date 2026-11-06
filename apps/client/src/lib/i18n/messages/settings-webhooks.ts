import type { Row } from '../row';

/** Settings: webhooks. */
export const settingsWebhooks = {
	'webhooks.title': ['Webhooks', 'Webhooks', 'Webhooks', 'Webhooks'],
	'webhooks.description': [
		'HTTP POST endpoints told immediately when a monitor goes down or recovers. Each delivery carries the same envelope Antenne receives, signed HMAC-SHA256 as sha256=<digest>.',
		'Des points d’accès HTTP POST prévenus immédiatement quand une sonde tombe ou se rétablit. Chaque envoi porte la même enveloppe qu’Antenne reçoit, signée HMAC-SHA256 sous la forme sha256=<digest>.',
		'Endpoints HTTP POST avisados al instante cuando un monitor cae o se recupera. Cada envío lleva el mismo sobre que recibe Antenne, firmado con HMAC-SHA256 como sha256=<digest>.',
		'HTTP-POST-Endpunkte, die sofort informiert werden, wenn ein Monitor ausfällt oder sich erholt. Jede Zustellung trägt denselben Umschlag wie bei Antenne, signiert mit HMAC-SHA256 als sha256=<digest>.'
	],
	'webhooks.add': ['Add webhook', 'Ajouter un webhook', 'Añadir webhook', 'Webhook hinzufügen'],
	'webhooks.empty.title': [
		'No webhooks configured',
		'Aucun webhook configuré',
		'Ningún webhook configurado',
		'Keine Webhooks konfiguriert'
	],
	'webhooks.empty.description': [
		'Point one at Antenne to get alerts wherever the suite already sends them, or at any endpoint that accepts JSON.',
		'Pointez-en un vers Antenne pour recevoir les alertes là où la suite les envoie déjà, ou vers tout point d’accès qui accepte du JSON.',
		'Apunta uno a Antenne para recibir alertas donde la suite ya las envía, o a cualquier endpoint que acepte JSON.',
		'Richten Sie einen auf Antenne, um Alarme dort zu erhalten, wohin die Suite sie bereits sendet, oder auf einen beliebigen Endpunkt, der JSON annimmt.'
	],
	'webhooks.empty.cta': [
		'Add your first webhook',
		'Ajouter votre premier webhook',
		'Añade tu primer webhook',
		'Ersten Webhook hinzufügen'
	],
	'webhooks.col.endpoint': ['Endpoint', 'Point d’accès', 'Endpoint', 'Endpunkt'],
	'webhooks.col.signature': ['Signature', 'Signature', 'Firma', 'Signatur'],
	'webhooks.col.state': ['State', 'État', 'Estado', 'Status'],
	'webhooks.unsigned': ['unsigned', 'non signé', 'sin firma', 'unsigniert'],
	'webhooks.test': ['Test', 'Tester', 'Probar', 'Testen'],
	'webhooks.testing': ['Testing…', 'Test en cours…', 'Probando…', 'Wird getestet…'],
	'webhooks.editAria': ['Edit {name}', 'Modifier {name}', 'Editar {name}', '{name} bearbeiten'],
	'webhooks.deleteAria': ['Delete {name}', 'Supprimer {name}', 'Eliminar {name}', '{name} löschen'],
	'webhooks.enableAria': ['Enable {name}', 'Activer {name}', 'Activar {name}', '{name} aktivieren'],
	'webhooks.toast.enabled': ['Webhook enabled.', 'Webhook activé.', 'Webhook activado.', 'Webhook aktiviert.'],
	'webhooks.toast.disabled': ['Webhook disabled.', 'Webhook désactivé.', 'Webhook desactivado.', 'Webhook deaktiviert.'],
	'webhooks.toast.deleted': ['Webhook deleted.', 'Webhook supprimé.', 'Webhook eliminado.', 'Webhook gelöscht.'],
	'webhooks.toast.updated': ['Webhook updated.', 'Webhook mis à jour.', 'Webhook actualizado.', 'Webhook aktualisiert.'],
	'webhooks.toast.added': ['Webhook added.', 'Webhook ajouté.', 'Webhook añadido.', 'Webhook hinzugefügt.'],
	'webhooks.error.load': [
		'Could not load webhooks.',
		'Impossible de charger les webhooks.',
		'No se pudieron cargar los webhooks.',
		'Webhooks konnten nicht geladen werden.'
	],
	'webhooks.error.update': [
		'Could not update webhook.',
		'Impossible de mettre à jour le webhook.',
		'No se pudo actualizar el webhook.',
		'Der Webhook konnte nicht aktualisiert werden.'
	],
	'webhooks.error.delete': [
		'Could not delete webhook.',
		'Impossible de supprimer le webhook.',
		'No se pudo eliminar el webhook.',
		'Der Webhook konnte nicht gelöscht werden.'
	],
	'webhooks.error.testFailed': ['Test failed.', 'Le test a échoué.', 'La prueba falló.', 'Test fehlgeschlagen.'],
	'webhooks.error.save': [
		'Could not save the webhook.',
		'Impossible d’enregistrer le webhook.',
		'No se pudo guardar el webhook.',
		'Der Webhook konnte nicht gespeichert werden.'
	],
	'webhooks.error.test': [
		'Could not test the webhook.',
		'Impossible de tester le webhook.',
		'No se pudo probar el webhook.',
		'Der Webhook konnte nicht getestet werden.'
	],
	'webhooks.testResult': [
		'HTTP {status} in {latency}',
		'HTTP {status} en {latency}',
		'HTTP {status} en {latency}',
		'HTTP {status} in {latency}'
	],
	'webhooks.testStatus': ['HTTP {status}', 'HTTP {status}', 'HTTP {status}', 'HTTP {status}'],
	'webhooks.delete.title': ['Delete webhook?', 'Supprimer le webhook ?', '¿Eliminar el webhook?', 'Webhook löschen?'],
	'webhooks.delete.description': [
		'Nothing else changes: the monitors keep running and the bus keeps publishing. This endpoint simply stops being told, and no alert already sent is recalled.',
		'Rien d’autre ne change : les sondes continuent de tourner et le bus de publier. Ce point d’accès n’est simplement plus prévenu, et aucune alerte déjà envoyée n’est rappelée.',
		'Nada más cambia: los monitores siguen funcionando y el bus sigue publicando. Este endpoint simplemente deja de recibir avisos, y ninguna alerta ya enviada se retira.',
		'Sonst ändert sich nichts: Die Monitore laufen weiter und der Bus veröffentlicht weiter. Dieser Endpunkt wird nur nicht mehr informiert, und bereits gesendete Alarme werden nicht zurückgerufen.'
	],
	'webhooks.delete.confirm': ['Delete webhook', 'Supprimer le webhook', 'Eliminar webhook', 'Webhook löschen'],

	'webhookDrawer.edit': ['Edit webhook', 'Modifier le webhook', 'Editar webhook', 'Webhook bearbeiten'],
	'webhookDrawer.namePlaceholder': [
		'e.g. Antenne Alert Hub',
		'p. ex. Hub d’alertes Antenne',
		'p. ej. Centro de alertas Antenne',
		'z. B. Antenne-Alarmzentrale'
	],
	'webhookDrawer.url': ['Webhook URL', 'URL du webhook', 'URL del webhook', 'Webhook-URL'],
	'webhookDrawer.urlHint': [
		'HTTP POST recipient for monitor.down and monitor.up alerts.',
		'Destinataire HTTP POST des alertes monitor.down et monitor.up.',
		'Destinatario HTTP POST de las alertas monitor.down y monitor.up.',
		'HTTP-POST-Empfänger für die Alarme monitor.down und monitor.up.'
	],
	'webhookDrawer.preset': ['Use Antenne preset', 'Utiliser le préréglage Antenne', 'Usar el ajuste de Antenne', 'Antenne-Vorgabe verwenden'],
	'webhookDrawer.secret': ['HMAC secret', 'Secret HMAC', 'Secreto HMAC', 'HMAC-Geheimnis'],
	'webhookDrawer.secretPlaceholder': [
		'Optional shared signing key',
		'Clé de signature partagée (facultatif)',
		'Clave de firma compartida (opcional)',
		'Optionaler gemeinsamer Signaturschlüssel'
	],
	'webhookDrawer.header': ['Signature header', 'En-tête de signature', 'Cabecera de firma', 'Signatur-Header'],
	'webhookDrawer.enable': ['Enable webhook', 'Activer le webhook', 'Activar webhook', 'Webhook aktivieren'],
	'webhookDrawer.enableHint': [
		'Send notifications on monitor state changes.',
		'Envoyer des notifications à chaque changement d’état d’une sonde.',
		'Enviar notificaciones cuando cambie el estado de un monitor.',
		'Benachrichtigungen bei Statuswechseln eines Monitors senden.'
	],
	'webhookDrawer.testEndpoint': ['Test endpoint', 'Tester le point d’accès', 'Probar endpoint', 'Endpunkt testen'],
	'webhookDrawer.urlFirst': [
		'Enter a webhook URL first.',
		'Saisissez d’abord une URL de webhook.',
		'Introduce primero una URL de webhook.',
		'Geben Sie zuerst eine Webhook-URL ein.'
	],
	'webhookDrawer.responded': [
		'Endpoint responded with HTTP {status} in {latency}',
		'Le point d’accès a répondu HTTP {status} en {latency}',
		'El endpoint respondió HTTP {status} en {latency}',
		'Der Endpunkt antwortete mit HTTP {status} in {latency}'
	],
	'webhookDrawer.returned': [
		'Endpoint returned HTTP {status}',
		'Le point d’accès a renvoyé HTTP {status}',
		'El endpoint devolvió HTTP {status}',
		'Der Endpunkt lieferte HTTP {status}'
	]
} satisfies Record<string, Row>;
