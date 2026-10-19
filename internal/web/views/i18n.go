package views

import "strings"

// Language is the set of display locales exposed by the Mini App. The users
// table intentionally keeps its small EN/RU enum for backwards compatibility;
// these extra locales live in the presentation cookie and never affect
// authorization, payments, or stored business data.
type Language string

const (
	LanguageRU Language = "ru"
	LanguageEN Language = "en"
	LanguageBE Language = "be"
	LanguageUK Language = "uk"
	LanguagePL Language = "pl"
)

type LanguageOption struct {
	Code       string
	NativeName string
	Name       string
}

var languageOptions = []LanguageOption{
	{Code: string(LanguageRU), NativeName: "Русский", Name: "Russian"},
	{Code: string(LanguageEN), NativeName: "English", Name: "English"},
	{Code: string(LanguageUK), NativeName: "Українська", Name: "Ukrainian"},
	{Code: string(LanguageBE), NativeName: "Беларуская", Name: "Belarusian"},
	{Code: string(LanguagePL), NativeName: "Polski", Name: "Polish"},
	{Code: "de", NativeName: "Deutsch", Name: "German"},
	{Code: "es", NativeName: "Español", Name: "Spanish"},
	{Code: "fr", NativeName: "Français", Name: "French"},
	{Code: "it", NativeName: "Italiano", Name: "Italian"},
	{Code: "pt", NativeName: "Português", Name: "Portuguese"},
	{Code: "tr", NativeName: "Türkçe", Name: "Turkish"},
	{Code: "ar", NativeName: "العربية", Name: "Arabic"},
	{Code: "id", NativeName: "Bahasa Indonesia", Name: "Indonesian"},
	{Code: "vi", NativeName: "Tiếng Việt", Name: "Vietnamese"},
	{Code: "ja", NativeName: "日本語", Name: "Japanese"},
	{Code: "ko", NativeName: "한국어", Name: "Korean"},
	{Code: "zh", NativeName: "中文", Name: "Chinese"},
	{Code: "nl", NativeName: "Nederlands", Name: "Dutch"},
	{Code: "cs", NativeName: "Čeština", Name: "Czech"},
	{Code: "sv", NativeName: "Svenska", Name: "Swedish"},
}

var supportedLanguages = func() map[string]struct{} {
	result := make(map[string]struct{}, len(languageOptions))
	for _, option := range languageOptions {
		result[option.Code] = struct{}{}
	}
	return result
}()

func NormalizeLanguage(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" || len(raw) > 16 {
		return string(LanguageRU)
	}
	// Telegram may send values such as en-US or uk_UA. Accept only a
	// well-formed two-letter base plus an optional region; arbitrary cookie
	// values are deliberately rejected and fall back to Russian.
	base := raw
	if index := strings.IndexAny(raw, "-_"); index >= 0 {
		base = raw[:index]
		region := raw[index+1:]
		if len(base) != 2 || len(region) < 2 || len(region) > 8 {
			return string(LanguageRU)
		}
		for _, r := range region {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
				return string(LanguageRU)
			}
		}
	}
	if len(base) != 2 {
		return string(LanguageRU)
	}
	for _, r := range base {
		if r < 'a' || r > 'z' {
			return string(LanguageRU)
		}
	}
	if _, ok := supportedLanguages[base]; ok {
		return base
	}
	return string(LanguageRU)
}

// TelegramLanguage returns the safe display locale for Telegram's hint. Keep
// this separate from the database enum so adding a UI locale never requires a
// migration of existing users.
func TelegramLanguage(raw string) string { return NormalizeLanguage(raw) }

func SupportedLanguages() []LanguageOption {
	result := make([]LanguageOption, len(languageOptions))
	copy(result, languageOptions)
	return result
}

func (d PageData) LanguageOptions() []LanguageOption {
	current := NormalizeLanguage(d.Language)
	result := make([]LanguageOption, 0, len(languageOptions))
	for _, option := range languageOptions {
		if option.Code != current {
			result = append(result, option)
		}
	}
	return result
}

func (d PageData) CurrentLanguageLabel() string {
	current := NormalizeLanguage(d.Language)
	for _, option := range languageOptions {
		if option.Code == current {
			return option.NativeName
		}
	}
	return "Русский"
}

func (d PageData) IsEnglish() bool { return NormalizeLanguage(d.Language) == string(LanguageEN) }

func (d PageData) LanguageCode() string { return NormalizeLanguage(d.Language) }

func (d PageData) OtherLanguage() string {
	if d.IsEnglish() {
		return string(LanguageRU)
	}
	return string(LanguageEN)
}

func (d PageData) OtherLanguageLabel() string {
	if d.IsEnglish() {
		return "Русский"
	}
	return "English"
}

// tr intentionally keeps translations in one place. The templates use keys
// rather than duplicating language conditionals, which makes adding a third
// locale a data-only change for the shared chrome.
func tr(language, key string) string {
	lang := NormalizeLanguage(language)
	if value, ok := translations[lang][key]; ok {
		return value
	}
	if lang != string(LanguageEN) {
		if value, ok := translations[string(LanguageEN)][key]; ok {
			// New locales intentionally fall back to English for a missing key;
			// this keeps a partially translated screen readable while the locale
			// grows, instead of mixing in an arbitrary key name.
			return value
		}
	}
	return translations[string(LanguageRU)][key]
}

var translations = map[string]map[string]string{
	string(LanguageRU): {
		"nav.aria": "Разделы студии", "nav.create": "Создать", "nav.library": "Материалы", "nav.order": "Заказ", "nav.admin": "Управление студией", "nav.language": "English",
		"page.title": "CONTGEN — студия видео", "page.footer": "Данные, которые хочется смотреть.",
		"welcome.eyebrow": "ИЗ ЦИФР — В ИСТОРИЮ", "welcome.title": "У данных\nесть движение.", "welcome.title.line1": "У данных", "welcome.title.line2": "есть движение", "welcome.lead": "Выбери данные и стиль. Посмотри, как твоя история будет выглядеть в кадре.", "welcome.kicker": "ТВОЯ СТУДИЯ", "welcome.local.title": "Локальный просмотр", "welcome.local.text": "Открой студию в браузере и проверь весь сценарий без Telegram-оплаты.", "welcome.telegram.title": "Подключаемся к Telegram", "welcome.telegram.text": "Открой приложение через бота, чтобы увидеть свои датасеты.", "welcome.local.button": "Открыть локальную студию", "welcome.retry": "Попробовать снова",
		"upload.title": "Название", "upload.file": "Нажми, чтобы выбрать файл", "upload.csv.hint": "CSV: year,entity,value. Значение — число от 0, период — целое число.", "upload.audio.hint": "MP3, WAV, OGG или M4A · до 20 МБ и 10 минут. Выбирай музыку, которую можно использовать в твоём видео.", "upload.example": "Скачать пример ↓", "upload.datasets": "Выбрать CSV", "upload.music": "Выбрать аудио", "upload.datasets.button": "Загрузить данные", "upload.music.button": "Загрузить музыку", "upload.admin.button": "Добавить в общую библиотеку",
		"library.kicker": "МАТЕРИАЛЫ", "library.title": "Твоя библиотека.", "library.lead": "Загружай данные и музыку. Общая коллекция доступна всем, личные материалы — только тебе.", "library.upload.datasets": "↑ Загрузить данные", "library.upload.music": "♫ Загрузить музыку", "library.datasets": "Датасеты", "library.music": "Музыка", "library.empty.datasets.title": "Начни с данных", "library.empty.datasets.text": "Загрузи CSV или скачай пример в форме выше.", "library.empty.music.title": "Задай настроение", "library.empty.music.text": "Загрузи любимый трек. Видео можно создать и без музыки.", "library.create": "Создать видео ↗", "library.common": "Общий", "library.personal": "Личный", "library.archived": "В архиве", "library.publish": "В общую библиотеку ↗", "library.delete.dataset": "Убрать датасет из библиотеки? Созданные видео сохранятся.", "library.delete.music": "Убрать трек из библиотеки? Созданные видео сохранятся.", "library.listen": "Послушать", "library.archive": "В архив", "library.restore": "Восстановить", "library.no.music": "Без музыки",
		"studio.kicker": "СОЗДАТЬ ВИДЕО", "studio.title": "Собери историю", "studio.title.muted": "из своих данных.", "studio.lead": "Выбери набор данных и стиль. Готовый ролик придёт отдельным сообщением в Telegram.", "studio.route.data": "Данные", "studio.route.style": "Стиль", "studio.route.telegram": "Telegram", "studio.data": "Данные", "studio.data.help": "Что показываем в ролике", "studio.no.datasets": "Пока нет наборов данных", "studio.no.datasets.text": "Открой материалы, чтобы загрузить CSV.", "studio.dataset": "Набор данных", "studio.style": "Стиль", "studio.style.help": "Визуальный характер истории", "studio.preview": "ПРЕДПРОСМОТР", "studio.preview.note": "Предпросмотр обновляется сразу после выбора настроек.", "studio.sound": "Звук", "studio.optional": "по желанию", "studio.sound.help": "Добавь атмосферу или оставь тишину", "studio.track": "Трек", "studio.no.music": "Без музыки", "studio.materials": "материалах", "studio.refresh": "Обновляем…", "studio.create": "Собрать заказ", "studio.ads": "2 рекламы / ролик", "studio.stars": "После оплаты Stars запускаем создание и отправляем видео в Telegram.", "studio.ads.help": "После двух просмотров ролика запускаем создание и отправляем видео в Telegram.", "studio.preview.empty": "Выбери датасет, чтобы увидеть его в кадре", "studio.preview.no.points": "В этом датасете пока нет точек", "studio.period": "ПЕРИОД", "studio.entities": "СУЩНОСТЕЙ", "studio.points": "ТОЧЕК", "studio.format": "Вертикальный формат · 9:16", "studio.audio": "Послушать выбранную музыку",
		"cart.kicker": "ЗАКАЗ", "cart.title": "Выбери способ оплаты.", "cart.lead": "Stars запускает ролики сразу. В Telegram — по две Reward-рекламы на каждый ролик.", "cart.empty.title": "Заказ пока пуст", "cart.empty.text": "Выбери данные и стиль в студии — готовая настройка появится здесь.", "cart.configure": "Настроить ролик", "cart.in.order": "В ЗАКАЗЕ", "cart.choose": "на выбор", "cart.overview.hint": "Проверь ролики перед оплатой", "cart.pay.choose": "Выбери способ оплаты", "cart.no.music": "Без музыки", "cart.remove": "Убрать из заказа", "cart.payment.aria": "Способ оплаты", "cart.ads": "2 рекламы / ролик", "cart.ads.free": "Бесплатно · всего %d просмотров", "cart.ads.disabled": "Скоро · рекламный провайдер ещё не подключён", "cart.stars": "Telegram Stars", "cart.stars.now": "сразу в очередь", "cart.ads.method": "СПОСОБ 1", "cart.ads.title": "Посмотри рекламу для каждого ролика", "cart.ads.current": "Ролик %d из %d · реклама %d из 2", "cart.ads.each": "На каждый ролик — два полных просмотра.", "cart.ad.first": "Первая реклама", "cart.ad.second": "Вторая реклама", "cart.ad.open": "Откроется поверх Mini App", "cart.ad.next": "После неё переходим к следующему ролику", "cart.ad.start": "Начать просмотр", "cart.ad.count": "%02d / %02d ↗", "cart.ad.foot": "Засчитывается только полностью просмотренная реклама. После двух просмотров ролика переходим к следующему.", "cart.stars.method": "СПОСОБ 2", "cart.stars.title": "Оплатить в Telegram", "cart.stars.text": "Откроется защищённое окно оплаты. После подтверждения ролик сразу попадёт в очередь.", "cart.pay": "Оплатить сейчас", "cart.promo.have": "У меня есть промокод", "cart.promo": "Промокод", "cart.promo.apply": "Применить промокод", "cart.foot": "Готовый MP4 придёт отдельным сообщением от бота.",
		"status.kicker": "ЗАКАЗ", "status.title": "Видео уже в пути.", "status.lead": "Готовый файл придёт отдельным сообщением в Telegram — открывать страницу и ждать не нужно.", "status.empty.title": "Заказов пока нет", "status.empty.text": "Настрой первый ролик — после двух реклам он появится здесь.", "status.create": "Создать ролик", "status.order": "ЗАКАЗ #%03d", "status.settings": "Настройки сохранены", "status.format": "Ролик в вертикальном формате 9:16", "status.telegram": "Готовый MP4 придёт сообщением от бота", "status.download": "Скачать MP4", "status.sent.title": "Ролик уже в чате", "status.sent.text": "Открой диалог с ботом — файл отправлен отдельным сообщением.", "status.retry.title": "Почти готово", "status.retry.text": "Видео создано. Telegram ещё принимает файл, экран обновится автоматически.", "status.retry.button": "Отправить ещё раз ↗", "status.failed": "Не получилось создать видео. Попробуй ещё раз.", "status.retry.create": "Повторить создание", "status.loading.pending": "Следующий ролик в очереди", "status.loading.pending.text": "Запустим подготовку автоматически — страницу можно закрыть", "status.loading.processing": "Собираем ролик", "status.loading.processing.text": "Анимация и музыка объединяются в готовый MP4", "status.loading.foot": "Обновляется автоматически",
		"order.cart": "Корзина", "order.paid": "Оформлен", "order.cancelled": "Отменён", "job.pending": "В очереди", "job.processing": "Создаём видео", "job.completed": "Готово", "job.failed": "Нужен повтор", "job.accepted": "Заказ принят", "job.pending.text": "Видео в очереди", "job.processing.text": "Создаём видео", "job.completed.text": "Видео создано", "job.failed.text": "Нужно повторить", "job.telegram": "Отправлено в Telegram", "job.message": "Готовим сообщение", "job.send": "Отправим в Telegram",
		"admin.dataset.empty.title": "Библиотека пуста", "admin.dataset.empty.text": "Добавь первый общий датасет.", "admin.music.empty.title": "Пока без музыки", "admin.music.empty.text": "Добавь трек для всех пользователей.", "admin.access": "ДОСТУП", "admin.operations": "ОПЕРАЦИИ", "admin.sales": "ПРОДАЖИ", "admin.all.users": "Все пользователи", "admin.accounts": "%d аккаунтов", "admin.discounts": "Скидки на видео", "admin.add.dataset": "↑ Общий датасет", "admin.add.music": "♫ Общая музыка", "admin.create.promo": "Создать промокод", "admin.today": "СЕГОДНЯ", "admin.summary": "Сводка студии", "admin.update": "Обновляется автоматически", "admin.users.count": "Пользователей", "admin.library.access": "Доступны всем пользователям", "admin.orders.count": "Заказов", "admin.jobs.count": "Видео создано", "admin.pending": "В очереди / в работе", "admin.paid": "ОПЛАЧЕНО", "admin.revenue": "Подтверждённые оплаты заказов.", "admin.retry": "Нужен повтор рендера", "admin.workflow": "КАК РАБОТАТЬ", "admin.workflow.title": "Рабочий процесс", "admin.kicker": "ПАНЕЛЬ УПРАВЛЕНИЯ", "admin.title": "Студия под контролем.", "admin.lead": "Материалы, пользователи, заказы и промокоды — в одном месте.", "admin.online": "Система онлайн", "admin.nav": "Разделы управления", "admin.overview": "Обзор", "admin.library": "Материалы", "admin.users": "Люди", "admin.orders": "Заказы", "admin.promos": "Промокоды",
	},
	string(LanguageEN): {
		"nav.aria": "Studio sections", "nav.create": "Create", "nav.library": "Materials", "nav.order": "Order", "nav.admin": "Studio admin", "nav.language": "Русский",
		"page.title": "CONTGEN — video studio", "page.footer": "Data worth watching.",
		"welcome.eyebrow": "FROM NUMBERS TO STORIES", "welcome.title": "Give your data\nmotion.", "welcome.title.line1": "Give your data", "welcome.title.line2": "motion", "welcome.lead": "Choose data and a style. See your story come alive in the frame.", "welcome.kicker": "YOUR STUDIO", "welcome.local.title": "Local preview", "welcome.local.text": "Open the studio in your browser and test the full flow without Telegram payments.", "welcome.telegram.title": "Connect to Telegram", "welcome.telegram.text": "Open the app through the bot to see your datasets.", "welcome.local.button": "Open local studio", "welcome.retry": "Try again",
		"upload.title": "Name", "upload.file": "Click to choose a file", "upload.csv.hint": "CSV: year,entity,value. Value must be a number, period an integer.", "upload.audio.hint": "MP3, WAV, OGG or M4A · up to 20 MB and 10 minutes. Use music you can legally include.", "upload.example": "Download example ↓", "upload.datasets": "Choose CSV", "upload.music": "Choose audio", "upload.datasets.button": "Upload data", "upload.music.button": "Upload music", "upload.admin.button": "Add to shared library",
		"library.kicker": "MATERIALS", "library.title": "Your library.", "library.lead": "Upload data and music. Shared materials are available to everyone; personal ones stay yours.", "library.upload.datasets": "↑ Upload data", "library.upload.music": "♫ Upload music", "library.datasets": "Datasets", "library.music": "Music", "library.empty.datasets.title": "Start with data", "library.empty.datasets.text": "Upload a CSV or download the example above.", "library.empty.music.title": "Set the mood", "library.empty.music.text": "Upload a track. You can also create a video without music.", "library.create": "Create video ↗", "library.common": "Shared", "library.personal": "Personal", "library.archived": "Archived", "library.publish": "Publish to shared library ↗", "library.delete.dataset": "Remove this dataset? Existing videos will stay.", "library.delete.music": "Remove this track? Existing videos will stay.", "library.listen": "Listen", "library.archive": "Archive", "library.restore": "Restore", "library.no.music": "No music",
		"studio.kicker": "CREATE VIDEO", "studio.title": "Build a story", "studio.title.muted": "from your data.", "studio.lead": "Choose a dataset and style. The finished video arrives as a separate Telegram message.", "studio.route.data": "Data", "studio.route.style": "Style", "studio.route.telegram": "Telegram", "studio.data": "Data", "studio.data.help": "What to show", "studio.no.datasets": "No datasets yet", "studio.no.datasets.text": "Open Materials to upload a CSV.", "studio.dataset": "Dataset", "studio.style": "Style", "studio.style.help": "Visual character of the story", "studio.preview": "PREVIEW", "studio.preview.note": "The preview updates as soon as you change a setting.", "studio.sound": "Sound", "studio.optional": "optional", "studio.sound.help": "Add atmosphere or leave it quiet", "studio.track": "Track", "studio.no.music": "No music", "studio.materials": "Materials", "studio.refresh": "Updating…", "studio.create": "Build order", "studio.ads": "2 ads / video", "studio.stars": "After a Stars payment we create the video and send it to Telegram.", "studio.ads.help": "After two ad views we create the video and send it to Telegram.", "studio.preview.empty": "Choose a dataset to see it in the frame", "studio.preview.no.points": "This dataset has no points yet", "studio.period": "PERIOD", "studio.entities": "ENTITIES", "studio.points": "POINTS", "studio.format": "Vertical format · 9:16", "studio.audio": "Listen to selected music",
		"cart.kicker": "ORDER", "cart.title": "Choose a payment method.", "cart.lead": "Stars starts videos right away. Telegram uses two Reward ads for each video.", "cart.empty.title": "Your order is empty", "cart.empty.text": "Choose data and a style in the studio — your setup will appear here.", "cart.configure": "Configure video", "cart.in.order": "IN ORDER", "cart.choose": "to choose", "cart.overview.hint": "Review videos before paying", "cart.pay.choose": "Choose a payment method", "cart.no.music": "No music", "cart.remove": "Remove from order", "cart.payment.aria": "Payment method", "cart.ads": "2 ads / video", "cart.ads.free": "Free · %d views total", "cart.ads.disabled": "Soon · the ad provider is not connected yet", "cart.stars": "Telegram Stars", "cart.stars.now": "queued immediately", "cart.ads.method": "METHOD 1", "cart.ads.title": "Watch an ad for each video", "cart.ads.current": "Video %d of %d · ad %d of 2", "cart.ads.each": "Each video needs two complete views.", "cart.ad.first": "First ad", "cart.ad.second": "Second ad", "cart.ad.open": "Opens over the Mini App", "cart.ad.next": "Then we move to the next video", "cart.ad.start": "Start watching", "cart.ad.count": "%02d / %02d ↗", "cart.ad.foot": "Only a fully watched ad counts. After two views we move to the next video.", "cart.stars.method": "METHOD 2", "cart.stars.title": "Pay in Telegram", "cart.stars.text": "A secure payment window will open. The video enters the queue after confirmation.", "cart.pay": "Pay now", "cart.promo.have": "I have a promo code", "cart.promo": "Promo code", "cart.promo.apply": "Apply promo code", "cart.foot": "The finished MP4 arrives as a separate bot message.",
		"status.kicker": "ORDER", "status.title": "Your video is on its way.", "status.lead": "The finished file arrives as a separate Telegram message — you can close this page.", "status.empty.title": "No orders yet", "status.empty.text": "Set up your first video — it will appear here after two ads.", "status.create": "Create video", "status.order": "ORDER #%03d", "status.settings": "Settings saved", "status.format": "Vertical 9:16 video", "status.telegram": "The finished MP4 arrives from the bot", "status.download": "Download MP4", "status.sent.title": "The video is in chat", "status.sent.text": "Open the bot chat — the file was sent as a separate message.", "status.retry.title": "Almost ready", "status.retry.text": "The video is ready. Telegram is still receiving the file; this screen updates automatically.", "status.retry.button": "Send again ↗", "status.failed": "The video could not be created. Try again.", "status.retry.create": "Create again", "status.loading.pending": "Next video in queue", "status.loading.pending.text": "We will start automatically — you can close this page", "status.loading.processing": "Creating your video", "status.loading.processing.text": "Animation and music are being combined into an MP4", "status.loading.foot": "Updates automatically",
		"order.cart": "Cart", "order.paid": "Placed", "order.cancelled": "Cancelled", "job.pending": "Queued", "job.processing": "Creating video", "job.completed": "Ready", "job.failed": "Retry needed", "job.accepted": "Order accepted", "job.pending.text": "Video is queued", "job.processing.text": "Creating video", "job.completed.text": "Video created", "job.failed.text": "Please retry", "job.telegram": "Sent to Telegram", "job.message": "Preparing message", "job.send": "Will send to Telegram",
		"admin.dataset.empty.title": "Library is empty", "admin.dataset.empty.text": "Add the first shared dataset.", "admin.music.empty.title": "No music yet", "admin.music.empty.text": "Add a track for all users.", "admin.access": "ACCESS", "admin.operations": "OPERATIONS", "admin.sales": "SALES", "admin.all.users": "All users", "admin.accounts": "%d accounts", "admin.discounts": "Video discounts", "admin.add.dataset": "↑ Shared dataset", "admin.add.music": "♫ Shared music", "admin.create.promo": "Create promo code", "admin.today": "TODAY", "admin.summary": "Studio summary", "admin.update": "Updates automatically", "admin.users.count": "Users", "admin.library.access": "Available to all users", "admin.orders.count": "Orders", "admin.jobs.count": "Videos created", "admin.pending": "Queued / processing", "admin.paid": "PAID", "admin.revenue": "Confirmed order payments.", "admin.retry": "Render retry needed", "admin.workflow": "HOW IT WORKS", "admin.workflow.title": "Workflow", "admin.kicker": "CONTROL PANEL", "admin.title": "Studio under control.", "admin.lead": "Materials, users, orders and promo codes in one place.", "admin.online": "System online", "admin.nav": "Admin sections", "admin.overview": "Overview", "admin.library": "Materials", "admin.users": "People", "admin.orders": "Orders", "admin.promos": "Promo codes",
	},
}

// The compact locale packs below cover the shared chrome and the primary
// checkout/status flow. Less frequently used copy falls back to English via
// tr, so adding a locale never produces a blank label or a translation key.
func init() {
	translations[string(LanguageRU)]["dataset.platform_sales"] = "Продажи игровых платформ · 2015–2024 (млн устройств)"
	translations[string(LanguageRU)]["dataset.streaming_genres"] = "Музыкальные жанры в стриминге · 2015–2024 (%)"
	translations[string(LanguageRU)]["dataset.renewable_growth"] = "Рост возобновляемой энергии · 2015–2024 (%)"
	translations[string(LanguageRU)]["dataset.population"] = "Население стран · 2015–2024 (млн человек)"
	translations[string(LanguageRU)]["dataset.co2"] = "Выбросы CO₂ на человека · 2015–2024 (тонн)"
	translations[string(LanguageRU)]["dataset.renewable_share"] = "Возобновляемая энергия · 2015–2024 (% финального потребления)"
	translations[string(LanguageEN)]["dataset.platform_sales"] = "Game platform sales · 2015–2024 (million devices)"
	translations[string(LanguageEN)]["dataset.streaming_genres"] = "Music genres in streaming · 2015–2024 (%)"
	translations[string(LanguageEN)]["dataset.renewable_growth"] = "Renewable energy growth · 2015–2024 (%)"
	translations[string(LanguageEN)]["dataset.population"] = "Population by country · 2015–2024 (million people)"
	translations[string(LanguageEN)]["dataset.co2"] = "CO₂ emissions per person · 2015–2024 (tonnes)"
	translations[string(LanguageEN)]["dataset.renewable_share"] = "Renewable energy · 2015–2024 (% of final consumption)"
	translations[string(LanguageUK)] = map[string]string{
		"nav.aria": "Розділи студії", "nav.create": "Створити", "nav.library": "Матеріали", "nav.order": "Замовлення", "nav.admin": "Керування студією", "nav.language": "Мова",
		"page.title": "CONTGEN — відеостудія", "page.footer": "Дані, які хочеться дивитися.",
		"studio.kicker": "СТВОРИТИ ВІДЕО", "studio.title": "Створи історію", "studio.title.muted": "зі своїх даних.", "studio.lead": "Обери набір даних і стиль. Готовий ролик прийде окремим повідомленням у Telegram.", "studio.data": "Дані", "studio.data.help": "Що показуємо в ролику", "studio.dataset": "Набір даних", "studio.style": "Стиль", "studio.style.help": "Візуальний характер історії", "studio.preview": "ПОПЕРЕДНІЙ ПЕРЕГЛЯД", "studio.sound": "Звук", "studio.optional": "за бажанням", "studio.track": "Трек", "studio.no.music": "Без музики", "studio.create": "Створити замовлення", "studio.ads": "2 реклами / ролик", "studio.period": "ПЕРІОД", "studio.entities": "СУТНОСТІ", "studio.points": "ТОЧКИ",
		"cart.kicker": "ЗАМОВЛЕННЯ", "cart.title": "Обери спосіб оплати.", "cart.lead": "Stars запускає ролики одразу. У Telegram — по дві Reward-реклами на кожен ролик.", "cart.empty.title": "Замовлення порожнє", "cart.configure": "Налаштувати ролик", "cart.ads": "2 реклами / ролик", "cart.ads.method": "СПОСІБ 1", "cart.ads.title": "Переглянь рекламу для кожного ролика", "cart.ads.current": "Ролик %d з %d · реклама %d з 2", "cart.ad.start": "Почати перегляд", "cart.stars.method": "СПОСІБ 2", "cart.stars.title": "Оплатити в Telegram", "cart.pay": "Оплатити зараз", "cart.foot": "Готовий MP4 прийде окремим повідомленням від бота.",
		"status.kicker": "ЗАМОВЛЕННЯ", "status.order": "ЗАМОВА #%03d", "status.title": "Відео вже в дорозі.", "status.lead": "Готовий файл прийде окремим повідомленням у Telegram — сторінку можна закрити.", "status.empty.title": "Замовлень ще немає", "status.create": "Створити ролик", "status.download": "Завантажити MP4", "status.sent.title": "Ролик уже в чаті", "status.sent.text": "Відкрий діалог із ботом — файл надіслано окремим повідомленням.",
		"order.cart": "Кошик", "order.paid": "Оформлено", "order.cancelled": "Скасовано", "job.pending": "У черзі", "job.processing": "Створюємо відео", "job.completed": "Готово", "job.failed": "Потрібен повтор",
	}
	translations[string(LanguageBE)] = map[string]string{
		"nav.aria": "Раздзелы студыі", "nav.create": "Стварыць", "nav.library": "Матэрыялы", "nav.order": "Замова", "nav.admin": "Кіраванне студыяй", "nav.language": "Мова",
		"page.title": "CONTGEN — відэастудыя", "page.footer": "Дадзеныя, якія хочацца глядзець.",
		"studio.kicker": "СТВАРЫЦЬ ВІДЭА", "studio.title": "Ствары гісторыю", "studio.title.muted": "са сваіх даных.", "studio.lead": "Выберы набор даных і стыль. Гатовы ролік прыйдзе асобным паведамленнем у Telegram.", "studio.data": "Даныя", "studio.data.help": "Што паказваем у роліку", "studio.dataset": "Набор даных", "studio.style": "Стыль", "studio.style.help": "Візуальны характар гісторыі", "studio.preview": "ПАПЯРЭДНІ ПРАГЛЯД", "studio.sound": "Гук", "studio.optional": "па жаданні", "studio.track": "Трэк", "studio.no.music": "Без музыкі", "studio.create": "Стварыць замову", "studio.ads": "2 рэкламы / ролік", "studio.period": "ПЕРЫЯД", "studio.entities": "СУТНАСЦІ", "studio.points": "ПУНКТЫ",
		"cart.kicker": "ЗАМОВА", "cart.title": "Выберы спосаб аплаты.", "cart.lead": "Stars запускае ролікі адразу. У Telegram — па дзве Reward-рэкламы на кожны ролік.", "cart.empty.title": "Замова пустая", "cart.configure": "Наладзіць ролік", "cart.ads": "2 рэкламы / ролік", "cart.ads.method": "СПОСАБ 1", "cart.ads.title": "Паглядзі рэкламу для кожнага роліка", "cart.ads.current": "Ролік %d з %d · рэклама %d з 2", "cart.ad.start": "Пачаць прагляд", "cart.stars.method": "СПОСАБ 2", "cart.stars.title": "Аплаціць у Telegram", "cart.pay": "Аплаціць зараз", "cart.foot": "Гатовы MP4 прыйдзе асобным паведамленнем ад бота.",
		"status.kicker": "ЗАМОВА", "status.order": "ЗАМОВА #%03d", "status.title": "Відэа ўжо ў дарозе.", "status.lead": "Гатовы файл прыйдзе асобным паведамленнем у Telegram — старонку можна закрыць.", "status.empty.title": "Замоў яшчэ няма", "status.create": "Стварыць ролік", "status.download": "Спампаваць MP4", "status.sent.title": "Ролік ужо ў чаце", "status.sent.text": "Адкрый дыялог з ботам — файл адпраўлены асобным паведамленнем.",
		"order.cart": "Кошык", "order.paid": "Аформлена", "order.cancelled": "Скасавана", "job.pending": "У чарзе", "job.processing": "Ствараем відэа", "job.completed": "Гатова", "job.failed": "Патрэбны паўтор",
	}
	translations[string(LanguagePL)] = map[string]string{
		"nav.aria": "Sekcje studia", "nav.create": "Utwórz", "nav.library": "Materiały", "nav.order": "Zamówienie", "nav.admin": "Panel studia", "nav.language": "Język",
		"page.title": "CONTGEN — studio wideo", "page.footer": "Dane, które warto oglądać.",
		"studio.kicker": "UTWÓRZ WIDEO", "studio.title": "Zbuduj historię", "studio.title.muted": "ze swoich danych.", "studio.lead": "Wybierz dane i styl. Gotowy film otrzymasz w osobnej wiadomości na Telegramie.", "studio.data": "Dane", "studio.data.help": "Co pokazujemy", "studio.dataset": "Zestaw danych", "studio.style": "Styl", "studio.style.help": "Charakter wizualny historii", "studio.preview": "PODGLĄD", "studio.sound": "Dźwięk", "studio.optional": "opcjonalnie", "studio.track": "Utwór", "studio.no.music": "Bez muzyki", "studio.create": "Utwórz zamówienie", "studio.ads": "2 reklamy / film", "studio.period": "OKRES", "studio.entities": "ELEMENTY", "studio.points": "PUNKTY",
		"cart.kicker": "ZAMÓWIENIE", "cart.title": "Wybierz sposób płatności.", "cart.lead": "Stars uruchamia filmy od razu. W Telegramie — po dwie reklamy Reward na każdy film.", "cart.empty.title": "Zamówienie jest puste", "cart.configure": "Skonfiguruj film", "cart.ads": "2 reklamy / film", "cart.ads.method": "SPOSÓB 1", "cart.ads.title": "Obejrzyj reklamę dla każdego filmu", "cart.ads.current": "Film %d z %d · reklama %d z 2", "cart.ad.start": "Rozpocznij oglądanie", "cart.stars.method": "SPOSÓB 2", "cart.stars.title": "Zapłać w Telegramie", "cart.pay": "Zapłać teraz", "cart.foot": "Gotowy MP4 przyjdzie w osobnej wiadomości od bota.",
		"status.kicker": "ZAMÓWIENIE", "status.order": "ZAMÓWIENIE #%03d", "status.title": "Twój film jest w drodze.", "status.lead": "Gotowy plik przyjdzie w osobnej wiadomości na Telegramie — możesz zamknąć stronę.", "status.empty.title": "Brak zamówień", "status.create": "Utwórz film", "status.download": "Pobierz MP4", "status.sent.title": "Film jest już na czacie", "status.sent.text": "Otwórz rozmowę z botem — plik został wysłany osobno.",
		"order.cart": "Koszyk", "order.paid": "Złożone", "order.cancelled": "Anulowane", "job.pending": "W kolejce", "job.processing": "Tworzymy film", "job.completed": "Gotowe", "job.failed": "Wymagana ponowna próba",
	}
	datasetTitles := map[string]map[string]string{
		string(LanguageUK): {
			"dataset.platform_sales": "Продажі ігрових платформ · 2015–2024 (млн пристроїв)", "dataset.streaming_genres": "Музичні жанри у стримінгу · 2015–2024 (%)", "dataset.renewable_growth": "Зростання відновлюваної енергії · 2015–2024 (%)", "dataset.population": "Населення країн · 2015–2024 (млн людей)", "dataset.co2": "Викиди CO₂ на людину · 2015–2024 (тонн)", "dataset.renewable_share": "Відновлювана енергія · 2015–2024 (% кінцевого споживання)",
		},
		string(LanguageBE): {
			"dataset.platform_sales": "Продаж гульнявых платформаў · 2015–2024 (млн прылад)", "dataset.streaming_genres": "Музычныя жанры ў стрымінгу · 2015–2024 (%)", "dataset.renewable_growth": "Рост аднаўляльнай энергіі · 2015–2024 (%)", "dataset.population": "Насельніцтва краін · 2015–2024 (млн чалавек)", "dataset.co2": "Выкіды CO₂ на чалавека · 2015–2024 (тон)", "dataset.renewable_share": "Аднаўляльная энергія · 2015–2024 (% фінальнага спажывання)",
		},
		string(LanguagePL): {
			"dataset.platform_sales": "Sprzedaż platform do gier · 2015–2024 (mln urządzeń)", "dataset.streaming_genres": "Gatunki muzyczne w streamingu · 2015–2024 (%)", "dataset.renewable_growth": "Rozwój energii odnawialnej · 2015–2024 (%)", "dataset.population": "Ludność krajów · 2015–2024 (mln osób)", "dataset.co2": "Emisje CO₂ na osobę · 2015–2024 (tony)", "dataset.renewable_share": "Energia odnawialna · 2015–2024 (% zużycia końcowego)",
		},
	}
	for language, values := range datasetTitles {
		for key, value := range values {
			translations[language][key] = value
		}
	}
}
