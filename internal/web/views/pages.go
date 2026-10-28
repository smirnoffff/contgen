package views

import (
	"fmt"
	"strings"

	"contgen/internal/catalog"
)

// DatasetTitle localizes only the built-in catalog. Personal CSV names are
// returned verbatim, while legacy built-in names still resolve to the stable
// catalog key until the next seed refreshes them.
func DatasetTitle(title, language string) string {
	return catalog.LocalizedTitle(title, NormalizeLanguage(language))
}

// PageData is presentation data shared by every authenticated screen.
type PageData struct {
	UserName   string
	Language   string
	IsAdmin    bool
	CartCount  int
	AdsEnabled bool
	Notice     string
	Error      string
}

type MusicOption struct {
	ID       int64
	Title    string
	URL      string
	Duration string
}

type CartItem struct {
	ID    int64
	Title string
	Theme string
	Music string
	Price string
}
type CartData struct {
	PageData
	OrderID         int64
	Items           []CartItem
	AdVideoCount    int
	AdTotal         int
	Subtotal        string
	Discount        string
	Total           string
	PromoCode       string
	CheckoutEnabled bool
}
type Job struct {
	ID              int64
	Title           string
	Theme           string
	Music           string
	Status          string
	CreatedAt       string
	Error           string
	VideoURL        string
	DownloadURL     string
	CanSendTelegram bool
	TelegramSent    bool
}
type Order struct {
	ID            int64
	DisplayNumber int64
	Status        string
	CreatedAt     string
	Total         string
	Payment       string
	Jobs          []Job
}
type OrdersData struct {
	PageData
	Orders    []Order
	HasActive bool
}
type DatasetItem struct {
	ID         int64
	Title      string
	Detail     string
	System     bool
	CanDelete  bool
	Active     bool
	CanPublish bool
}
type MusicItem struct {
	ID         int64
	Title      string
	Detail     string
	URL        string
	System     bool
	Active     bool
	CanDelete  bool
	CanPublish bool
}
type LibraryData struct {
	PageData
	Datasets []DatasetItem
	Music    []MusicItem
}
type AdminUser struct {
	ID         int64
	Name       string
	Username   string
	TelegramID string
	Role       string
	JoinedAt   string
	IsSelf     bool
}
type PromoItem struct {
	ID       int64
	Code     string
	Discount string
	Usage    string
	Expires  string
	Active   bool
	State    string
}
type AdminOrder struct {
	Order
	OwnerName     string
	OwnerUsername string
	TelegramID    string
	LocalDemo     bool
	ItemCount     int64
}

type AdminData struct {
	PageData
	Tab             string
	UserCount       int64
	OrderCount      int64
	JobCount        int64
	PendingCount    int64
	FailedCount     int64
	Revenue         string
	StarsPayments   int64
	AdOrders        int64
	PromoOrders     int64
	CartCountTotal  int64
	DeliveredCount  int64
	QueuedCount     int64
	ProcessingCount int64
	IncludeDemo     bool
	UpdatedAt       string
	FilterStatus    string
	Search          string
	CurrentPage     int
	TotalOrders     int64
	HasPrevious     bool
	HasNext         bool
	Users           []AdminUser
	Orders          []AdminOrder
	Datasets        []DatasetItem
	Music           []MusicItem
	Promos          []PromoItem
}

func datasetDisplayTitle(language, title string, system bool) string {
	if !system {
		return title
	}
	return catalog.LocalizedTitle(title, NormalizeLanguage(language))
}

func routeID(prefix string, id int64, suffix string) string {
	return fmt.Sprintf("%s/%d%s", prefix, id, suffix)
}
func jobLabel(status, language string) string {
	key := "job." + strings.ToLower(status)
	return tr(language, key)
}
func orderLabel(status, language string) string {
	if value := tr(language, "order."+strings.ToLower(status)); value != "" {
		return value
	}
	return status
}
func orderNumber(order Order) int64 {
	if order.DisplayNumber > 0 {
		return order.DisplayNumber
	}
	return order.ID
}
func statusClass(status string) string   { return "status status-" + strings.ToLower(status) }
func jobStateClass(status string) string { return "job-state-initial-" + strings.ToLower(status) }

func jobStepClass(job Job, step int) string {
	switch step {
	case 1:
		return "is-done"
	case 2:
		switch job.Status {
		case "PENDING", "PROCESSING":
			return "is-active"
		case "COMPLETED":
			return "is-done"
		case "FAILED":
			return "is-error"
		}
	case 3:
		if job.TelegramSent {
			return "is-done"
		}
		if job.Status == "COMPLETED" {
			return "is-active"
		}
	}
	return ""
}

func jobStepIcon(job Job, step int) string {
	if jobStepClass(job, step) == "is-done" {
		return "✓"
	}
	return fmt.Sprintf("%02d", step)
}

func jobStepText(job Job, step int, language string) string {
	switch step {
	case 1:
		return tr(language, "job.accepted")
	case 2:
		switch job.Status {
		case "PENDING":
			return tr(language, "job.pending.text")
		case "PROCESSING":
			return tr(language, "job.processing.text")
		case "COMPLETED":
			return tr(language, "job.completed.text")
		case "FAILED":
			return tr(language, "job.failed.text")
		}
		return tr(language, "job.processing.text")
	case 3:
		if job.TelegramSent {
			return tr(language, "job.telegram")
		}
		if job.Status == "COMPLETED" {
			return tr(language, "job.message")
		}
		return tr(language, "job.send")
	}
	return ""
}
func (d StudioData) PageInfo() PageData {
	p := d.PageData
	if p.UserName == "" {
		p.UserName = d.User.FirstName
	}
	p.IsAdmin = p.IsAdmin || string(d.User.Role) == "ADMIN"
	return p
}
func (d StudioData) SelectedMusicURL() string {
	for _, m := range d.Music {
		if m.ID == d.MusicID {
			return m.URL
		}
	}
	return ""
}

func currentPage(active, section string) string {
	if active == section {
		return "page"
	}
	return "false"
}

func orderPath(d PageData) string {
	if d.CartCount > 0 {
		return "/cart"
	}
	return "/status"
}

func orderCurrent(active string) string {
	if active == "cart" || active == "status" {
		return "page"
	}
	return "false"
}
func uploadRoute(kind string, admin bool) string {
	if admin {
		return "/admin/" + kind + "/upload"
	}
	return "/" + kind + "/upload"
}
func uploadID(kind string, admin bool) string {
	if admin {
		return "admin-" + kind
	}
	return "upload-" + kind
}
func uploadPlaceholder(kind, language string) string {
	if kind == "datasets" {
		if NormalizeLanguage(language) == string(LanguageEN) {
			return "For example, language popularity"
		}
		return "Например, популярность языков"
	}
	if NormalizeLanguage(language) == string(LanguageEN) {
		return "For example, evening synthwave"
	}
	return "Например, вечерний синтвейв"
}
func uploadLabel(kind, language string) string {
	if kind == "datasets" {
		return tr(language, "upload.datasets")
	}
	return tr(language, "upload.music")
}
func uploadAccept(kind string) string {
	if kind == "datasets" {
		return ".csv,text/csv"
	}
	return ".mp3,.wav,.ogg,.m4a,audio/*"
}
func uploadButton(kind string, admin bool, language string) string {
	if admin {
		return tr(language, "upload.admin.button")
	}
	if kind == "datasets" {
		return tr(language, "upload.datasets.button")
	}
	return tr(language, "upload.music.button")
}

func jobPollURL(id int64, admin bool) string {
	if admin {
		return routeID("/admin/jobs", id, "")
	}
	return routeID("/jobs", id, "")
}
func jobPollTrigger(status string) string {
	if status == "PENDING" || status == "PROCESSING" {
		return "every 3s"
	}
	return "none"
}
func jobActionURL(id int64, admin bool, action string) string {
	if admin {
		return routeID("/admin/jobs", id, "/"+action)
	}
	return routeID("/jobs", id, "/"+action)
}

func archiveLabel(active bool) string {
	if active {
		return "В архив"
	}
	return "Восстановить"
}

func ternaryKey(condition bool, whenTrue, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func cartCountLabel(language string, count int) string {
	if NormalizeLanguage(language) == string(LanguageEN) {
		if count == 1 {
			return fmt.Sprintf("%d video", count)
		}
		return fmt.Sprintf("%d videos", count)
	}
	if count%10 == 1 && count%100 != 11 {
		return fmt.Sprintf("%d ролик", count)
	}
	if count%10 >= 2 && count%10 <= 4 && (count%100 < 10 || count%100 >= 20) {
		return fmt.Sprintf("%d ролика", count)
	}
	return fmt.Sprintf("%d роликов", count)
}

type adminTabOption struct{ Code, Label string }

func adminTabs(language string) []adminTabOption {
	return []adminTabOption{{"overview", tr(language, "admin.overview")}, {"library", tr(language, "admin.library")}, {"users", tr(language, "admin.users")}, {"orders", tr(language, "admin.orders")}, {"promos", tr(language, "admin.promos")}}
}
func adminTab(tab string) string {
	if tab == "" {
		return "overview"
	}
	return tab
}
func initial(name string) string {
	for _, r := range name {
		return strings.ToUpper(string(r))
	}
	return "·"
}
