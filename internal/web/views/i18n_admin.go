package views

func init() {
	labels := map[string]map[string]string{
		"ru": {
			"admin.notice.music.updated":     "Статус музыки обновлён",
			"admin.notice.dataset.published": "Датасет опубликован для всех",
			"admin.notice.music.published":   "Музыка опубликована для всех",
			"admin.notice.job.queued":        "Видео снова поставлено в очередь",
			"admin.error.role":               "Некорректная роль",
			"admin.error.self.demote":        "Нельзя снять роль у самого себя",
			"admin.notice.role.updated":      "Роль обновлена",
			"admin.error.promo":              "Проверь код и скидку",
			"admin.error.limit":              "Некорректный лимит",
			"admin.error.date":               "Некорректная дата",
			"admin.notice.promo.created":     "Промокод создан",
			"admin.notice.promo.updated":     "Статус промокода обновлён",

			"admin.title": "Панель студии.", "admin.lead": "Заказы, материалы и доступ — всё для работы студии.",
			"admin.scope.real": "Только Telegram", "admin.scope.all": "Включая тестовые данные", "admin.scope.demo.hint": "Включены локальные тестовые аккаунты и их заказы. Нажми на переключатель, чтобы вернуться к данным Telegram.",
			"admin.refresh": "Обновить", "admin.period.all": "ЗА ВСЁ ВРЕМЯ", "admin.snapshot": "На", "admin.stars.received": "ПОЛУЧЕНО TELEGRAM STARS", "admin.stars.payments": "Успешных платежей: %d", "admin.stars.hint": "Только успешные платежи Stars. Заказы за рекламу и промокоды не прибавляют звёзды. Это поступления, а не баланс кошелька.",
			"admin.ad.orders": "Заказов за рекламу", "admin.promo.orders": "Заказов по промокоду", "admin.orders.placed": "Оформлено заказов", "admin.delivered": "Отправлено в Telegram", "admin.queue": "ОЧЕРЕДЬ ГЕНЕРАЦИИ", "admin.overview.hint": "Счётчики генерации учитывают только оформленные заказы. Корзины не входят в очередь. Данные обновляются по кнопке «Обновить».",
			"admin.library.title": "Библиотека студии", "admin.library.hint": "Общие и личные материалы", "admin.list.limit": "Показаны последние 100 записей.", "admin.self": "Твой аккаунт · администратор", "admin.role": "Роль", "admin.role.user": "Пользователь", "admin.role.admin": "Администратор", "admin.save": "Сохранить",
			"admin.orders.found": "Найдено: %d", "admin.search": "Имя, @username, Telegram ID или ID заказа", "admin.search.button": "Найти", "admin.filter.label": "Статус заказа", "admin.filter.placed": "Оформленные", "admin.filter.carts": "Корзины", "admin.filter.all": "Все", "admin.orders.empty": "Заказов не найдено", "admin.orders.empty.hint": "Попробуй другой поиск или выбери другой статус.", "admin.pages": "Страницы заказов", "admin.page": "Страница %d", "admin.previous": "Назад", "admin.next": "Далее",
			"admin.payment.unpaid": "Не оплачен", "admin.payment.ads": "За просмотр рекламы · 0 Stars", "admin.payment.promo": "Промокод · 0 Stars", "admin.payment.unknown": "Способ оплаты не указан", "admin.test": "Тестовый", "admin.order.videos": "Роликов: %d", "admin.order.details": "Ролики и состояние", "admin.video.sent": "В Telegram ✓", "admin.video.unsent": "Ещё не отправлен",
			"admin.promo.code": "Код", "admin.promo.discount": "Скидка, %", "admin.promo.limit": "Лимит использований", "admin.promo.unlimited": "Без лимита", "admin.promo.expiry": "Действует до · необязательно", "admin.promo.used": "Использовано:", "admin.promo.no.expiry": "Без срока действия", "admin.promo.until": "До %s", "admin.promo.state.active": "Доступен", "admin.promo.state.disabled": "Отключён", "admin.promo.state.expired": "Истёк срок", "admin.promo.state.exhausted": "Лимит исчерпан", "admin.enable": "Включить", "admin.disable": "Отключить", "admin.promos.empty": "Промокодов пока нет", "admin.promos.empty.hint": "Создай скидку с ограничением по сроку или количеству использований.",
			"admin.retry.unavailable": "Повтор недоступен: ролик не найден или его состояние уже изменилось.",
		},
		"en": {
			"admin.notice.music.updated":     "Music status updated",
			"admin.notice.dataset.published": "Dataset published for everyone",
			"admin.notice.music.published":   "Music published for everyone",
			"admin.notice.job.queued":        "Video queued again",
			"admin.error.role":               "Invalid role",
			"admin.error.self.demote":        "You cannot remove your own administrator role",
			"admin.notice.role.updated":      "Role updated",
			"admin.error.promo":              "Check the code and discount",
			"admin.error.limit":              "Invalid usage limit",
			"admin.error.date":               "Invalid date",
			"admin.notice.promo.created":     "Promo code created",
			"admin.notice.promo.updated":     "Promo code status updated",

			"admin.title": "Studio dashboard.", "admin.lead": "Orders, materials and access — your studio at a glance.",
			"admin.scope.real": "Telegram users only", "admin.scope.all": "Including test data", "admin.scope.demo.hint": "Local test accounts and their orders are included. Use the switch to return to Telegram data.",
			"admin.refresh": "Refresh", "admin.period.all": "ALL TIME", "admin.snapshot": "As of", "admin.stars.received": "TELEGRAM STARS RECEIVED", "admin.stars.payments": "Successful payments: %d", "admin.stars.hint": "Successful Stars payments only. Ad views and promo codes do not add Stars. This is received revenue, not a wallet balance.",
			"admin.ad.orders": "Orders via ads", "admin.promo.orders": "Orders via promo codes", "admin.orders.placed": "Orders placed", "admin.delivered": "Sent to Telegram", "admin.queue": "RENDER QUEUE", "admin.overview.hint": "Video counters include placed orders only. Carts do not enter the render queue. Use Refresh to update the figures.",
			"admin.library.title": "Studio library", "admin.library.hint": "Shared and personal materials", "admin.list.limit": "Showing the latest 100 records.", "admin.self": "Your account · administrator", "admin.role": "Role", "admin.role.user": "User", "admin.role.admin": "Administrator", "admin.save": "Save",
			"admin.orders.found": "Found: %d", "admin.search": "Name, @username, Telegram ID or order ID", "admin.search.button": "Search", "admin.filter.label": "Order status", "admin.filter.placed": "Placed", "admin.filter.carts": "Carts", "admin.filter.all": "All", "admin.orders.empty": "No orders found", "admin.orders.empty.hint": "Try a different search or choose another status.", "admin.pages": "Order pages", "admin.page": "Page %d", "admin.previous": "Previous", "admin.next": "Next",
			"admin.payment.unpaid": "Not paid", "admin.payment.ads": "Ad views · 0 Stars", "admin.payment.promo": "Promo code · 0 Stars", "admin.payment.unknown": "Payment method missing", "admin.test": "Test", "admin.order.videos": "Videos: %d", "admin.order.details": "Videos and status", "admin.video.sent": "In Telegram ✓", "admin.video.unsent": "Not sent yet",
			"admin.promo.code": "Code", "admin.promo.discount": "Discount, %", "admin.promo.limit": "Usage limit", "admin.promo.unlimited": "Unlimited", "admin.promo.expiry": "Valid until · optional", "admin.promo.used": "Used:", "admin.promo.no.expiry": "No expiry date", "admin.promo.until": "Until %s", "admin.promo.state.active": "Available", "admin.promo.state.disabled": "Disabled", "admin.promo.state.expired": "Expired", "admin.promo.state.exhausted": "Limit reached", "admin.enable": "Enable", "admin.disable": "Disable", "admin.promos.empty": "No promo codes yet", "admin.promos.empty.hint": "Create a discount with an expiry date or usage limit.",
			"admin.retry.unavailable": "Cannot retry: the video was not found or its status has already changed.",
		},
		"de": {
			"admin.notice.music.updated":     "Musikstatus aktualisiert",
			"admin.notice.dataset.published": "Datensatz für alle veröffentlicht",
			"admin.notice.music.published":   "Musik für alle veröffentlicht",
			"admin.notice.job.queued":        "Video erneut eingereiht",
			"admin.error.role":               "Ungültige Rolle",
			"admin.error.self.demote":        "Du kannst deine eigene Administratorrolle nicht entfernen",
			"admin.notice.role.updated":      "Rolle aktualisiert",
			"admin.error.promo":              "Prüfe Code und Rabatt",
			"admin.error.limit":              "Ungültiges Nutzungslimit",
			"admin.error.date":               "Ungültiges Datum",
			"admin.notice.promo.created":     "Promo-Code erstellt",
			"admin.notice.promo.updated":     "Promo-Code-Status aktualisiert",

			"admin.title": "Studioübersicht.", "admin.lead": "Bestellungen, Materialien und Zugriff auf einen Blick.",
			"admin.scope.real": "Nur Telegram-Nutzer", "admin.scope.all": "Mit Testdaten", "admin.scope.demo.hint": "Lokale Testkonten und ihre Bestellungen werden mitgezählt. Mit dem Schalter kehrst du zu Telegram-Daten zurück.",
			"admin.refresh": "Aktualisieren", "admin.period.all": "GESAMTER ZEITRAUM", "admin.snapshot": "Stand", "admin.stars.received": "ERHALTENE TELEGRAM STARS", "admin.stars.payments": "Erfolgreiche Zahlungen: %d", "admin.stars.hint": "Nur erfolgreiche Stars-Zahlungen. Anzeigen und Promo-Codes bringen keine Stars. Dies sind Einnahmen, kein Wallet-Guthaben.",
			"admin.ad.orders": "Bestellungen über Anzeigen", "admin.promo.orders": "Bestellungen über Promo-Codes", "admin.orders.placed": "Aufgegebene Bestellungen", "admin.delivered": "An Telegram gesendet", "admin.queue": "RENDER-WARTESCHLANGE", "admin.overview.hint": "Videozähler berücksichtigen nur aufgegebene Bestellungen. Warenkörbe stehen nicht in der Warteschlange. Mit Aktualisieren werden die Werte neu geladen.",
			"admin.library.title": "Studiobibliothek", "admin.library.hint": "Gemeinsame und persönliche Materialien", "admin.list.limit": "Die letzten 100 Einträge werden angezeigt.", "admin.self": "Dein Konto · Administrator", "admin.role": "Rolle", "admin.role.user": "Nutzer", "admin.role.admin": "Administrator", "admin.save": "Speichern",
			"admin.orders.found": "Gefunden: %d", "admin.search": "Name, @Nutzername, Telegram-ID oder Bestell-ID", "admin.search.button": "Suchen", "admin.filter.label": "Bestellstatus", "admin.filter.placed": "Aufgegeben", "admin.filter.carts": "Warenkörbe", "admin.filter.all": "Alle", "admin.orders.empty": "Keine Bestellungen gefunden", "admin.orders.empty.hint": "Versuche eine andere Suche oder einen anderen Status.", "admin.pages": "Bestellseiten", "admin.page": "Seite %d", "admin.previous": "Zurück", "admin.next": "Weiter",
			"admin.payment.unpaid": "Nicht bezahlt", "admin.payment.ads": "Anzeigen angesehen · 0 Stars", "admin.payment.promo": "Promo-Code · 0 Stars", "admin.payment.unknown": "Zahlungsart fehlt", "admin.test": "Test", "admin.order.videos": "Videos: %d", "admin.order.details": "Videos und Status", "admin.video.sent": "In Telegram ✓", "admin.video.unsent": "Noch nicht gesendet",
			"admin.promo.code": "Code", "admin.promo.discount": "Rabatt, %", "admin.promo.limit": "Nutzungslimit", "admin.promo.unlimited": "Unbegrenzt", "admin.promo.expiry": "Gültig bis · optional", "admin.promo.used": "Verwendet:", "admin.promo.no.expiry": "Ohne Ablaufdatum", "admin.promo.until": "Bis %s", "admin.promo.state.active": "Verfügbar", "admin.promo.state.disabled": "Deaktiviert", "admin.promo.state.expired": "Abgelaufen", "admin.promo.state.exhausted": "Limit erreicht", "admin.enable": "Aktivieren", "admin.disable": "Deaktivieren", "admin.promos.empty": "Noch keine Promo-Codes", "admin.promos.empty.hint": "Erstelle einen Rabatt mit Ablaufdatum oder Nutzungslimit.",
			"admin.retry.unavailable": "Kein neuer Versuch möglich: Video nicht gefunden oder Status bereits geändert.",
		},
	}
	for language, values := range labels {
		for key, value := range values {
			translations[language][key] = value
		}
	}
}
