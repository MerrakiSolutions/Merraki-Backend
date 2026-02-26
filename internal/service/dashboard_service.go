package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ActivityLog mirrors the frontend ActivityLog type.
type ActivityLog struct {
	ID        string    `json:"id"`
	AdminName string    `json:"adminName"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	IP        string    `json:"ip,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type DashboardService struct {
	db *pgxpool.Pool
}

func NewDashboardService(db *pgxpool.Pool) *DashboardService {
	return &DashboardService{db: db}
}

// GetStats returns the full payload expected by the frontend DashboardStats type.
func (s *DashboardService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// ── Orders ───────────────────────────────────────────────────────────────
	// Revenue uses total_inr (INR is the base currency for this platform)
	var totalOrders int
	var totalRevenue float64
	_ = s.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total_inr), 0)
		FROM orders
	`).Scan(&totalOrders, &totalRevenue)

	// Pending = approved_by IS NULL (awaiting approval)
	var pendingOrders int
	_ = s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM orders
		WHERE approved_by IS NULL AND status NOT IN ('rejected', 'cancelled')
	`).Scan(&pendingOrders)

	// Completed today = completed_at is today
	var completedToday int
	_ = s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM orders
		WHERE DATE(completed_at) = CURRENT_DATE
	`).Scan(&completedToday)

	result["totalOrders"] = totalOrders
	result["pendingOrders"] = pendingOrders
	result["totalRevenue"] = totalRevenue
	result["revenueGrowth"] = 0.0

	// ── Orders by status (pie chart) ─────────────────────────────────────────
	statusRows, err := s.db.Query(ctx, `
		SELECT status, COUNT(*) AS count
		FROM orders
		GROUP BY status
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("ordersByStatus query: %w", err)
	}
	defer statusRows.Close()

	ordersByStatus := []map[string]interface{}{}
	for statusRows.Next() {
		var status string
		var count int
		if err := statusRows.Scan(&status, &count); err != nil {
			continue
		}
		ordersByStatus = append(ordersByStatus, map[string]interface{}{
			"status": status,
			"count":  count,
		})
	}
	result["ordersByStatus"] = ordersByStatus

	// ── Monthly revenue — last 12 months (composed chart) ────────────────────
	monthRows, err := s.db.Query(ctx, `
		SELECT
			TO_CHAR(DATE_TRUNC('month', created_at), 'Mon ''YY') AS month,
			COALESCE(SUM(total_inr), 0)::float                   AS revenue,
			COUNT(*)::int                                          AS orders
		FROM orders
		WHERE created_at >= NOW() - INTERVAL '12 months'
		GROUP BY DATE_TRUNC('month', created_at)
		ORDER BY DATE_TRUNC('month', created_at)
	`)
	if err != nil {
		return nil, fmt.Errorf("monthlyRevenue query: %w", err)
	}
	defer monthRows.Close()

	monthlyRevenue := []map[string]interface{}{}
	for monthRows.Next() {
		var month string
		var revenue float64
		var orders int
		if err := monthRows.Scan(&month, &revenue, &orders); err != nil {
			continue
		}
		monthlyRevenue = append(monthlyRevenue, map[string]interface{}{
			"month":   month,
			"revenue": revenue,
			"orders":  orders,
		})
	}
	result["monthlyRevenue"] = monthlyRevenue

	// ── Daily revenue heatmap — last 84 days ─────────────────────────────────
	dayRows, err := s.db.Query(ctx, `
		SELECT
			TO_CHAR(d.day, 'YYYY-MM-DD')            AS date,
			COALESCE(SUM(o.total_inr), 0)::float    AS value
		FROM generate_series(
			CURRENT_DATE - INTERVAL '83 days',
			CURRENT_DATE,
			'1 day'::interval
		) AS d(day)
		LEFT JOIN orders o ON DATE(o.created_at) = d.day
		GROUP BY d.day
		ORDER BY d.day
	`)
	if err != nil {
		return nil, fmt.Errorf("dailyRevenue query: %w", err)
	}
	defer dayRows.Close()

	dailyRevenue := []map[string]interface{}{}
	for dayRows.Next() {
		var date string
		var value float64
		if err := dayRows.Scan(&date, &value); err != nil {
			continue
		}
		dailyRevenue = append(dailyRevenue, map[string]interface{}{
			"date":  date,
			"value": value,
		})
	}
	result["dailyRevenue"] = dailyRevenue

	// ── Blog ─────────────────────────────────────────────────────────────────
	var totalPosts, publishedPosts int
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM blog_posts`).Scan(&totalPosts)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM blog_posts WHERE status = 'published'`).Scan(&publishedPosts)
	result["totalBlogPosts"] = totalPosts
	result["publishedBlogPosts"] = publishedPosts

	// ── Templates ────────────────────────────────────────────────────────────
	var totalTemplates, activeTemplates int
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM templates`).Scan(&totalTemplates)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM templates WHERE is_active = true`).Scan(&activeTemplates)
	result["totalTemplates"] = totalTemplates
	result["activeTemplates"] = activeTemplates

	// ── Newsletter ───────────────────────────────────────────────────────────
	var totalSubs, activeSubs int
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM newsletter_subscribers`).Scan(&totalSubs)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM newsletter_subscribers WHERE status = 'active'`).Scan(&activeSubs)
	result["newsletter"] = map[string]interface{}{
		"total":  totalSubs,
		"active": activeSubs,
	}

	// ── Contacts ─────────────────────────────────────────────────────────────
	var totalContacts, newContacts int
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM contacts`).Scan(&totalContacts)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM contacts WHERE status = 'new'`).Scan(&newContacts)
	result["contacts"] = map[string]interface{}{
		"total": totalContacts,
		"new":   newContacts,
	}

	// ── Admins ───────────────────────────────────────────────────────────────
	var totalAdmins, activeAdmins int
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM admins`).Scan(&totalAdmins)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM admins WHERE is_active = true`).Scan(&activeAdmins)
	result["totalUsers"] = totalAdmins
	result["activeUsers"] = activeAdmins
	result["userGrowth"] = 0.0

	// ── Conversion funnel ────────────────────────────────────────────────────
	// visitors  → newsletter subscribers (best proxy without analytics table)
	// signups   → newsletter subscribers
	// testCompleted → test_submissions completed
	// purchased → orders with completed_at set (i.e. fulfilled)
	var testSubmissions, purchasedOrders int
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM test_submissions`).Scan(&testSubmissions)
	_ = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE completed_at IS NOT NULL`).Scan(&purchasedOrders)

	result["conversionFunnel"] = map[string]interface{}{
		"visitors":      totalSubs,
		"signups":       totalSubs,
		"testCompleted": testSubmissions,
		"purchased":     purchasedOrders,
	}

	// ── Sparklines — last 7 days per metric ──────────────────────────────────
	result["sparklines"] = map[string]interface{}{
		"users":     s.sparklineCounts(ctx, "admins", "created_at", "", 7),
		"orders":    s.sparklineCounts(ctx, "orders", "created_at", "", 7),
		"revenue":   s.sparklineSum(ctx, "orders", "total_inr", "created_at", 7),
		"pending":   s.sparklineCounts(ctx, "orders", "created_at", "approved_by IS NULL AND status NOT IN ('rejected','cancelled')", 7),
		"posts":     s.sparklineCounts(ctx, "blog_posts", "created_at", "", 7),
		"templates": s.sparklineCounts(ctx, "templates", "created_at", "", 7),
	}

	return result, nil
}

// GetSummary is an alias for GetStats (kept for backward compat with GetSummary callers).
func (s *DashboardService) GetSummary(ctx context.Context) (map[string]interface{}, error) {
	return s.GetStats(ctx)
}

// GetActivity returns the last 50 rows from activity_logs joined with admins.
func (s *DashboardService) GetActivity(ctx context.Context) ([]ActivityLog, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			al.id::text,
			COALESCE(a.name, 'System')                                    AS admin_name,
			al.action,
			COALESCE(al.entity_type, '')
				|| CASE WHEN al.entity_id IS NOT NULL
					THEN ' #' || al.entity_id::text
					ELSE ''
				   END                                                     AS resource,
			COALESCE(al.ip_address::text, '')                             AS ip,
			al.created_at
		FROM activity_logs al
		LEFT JOIN admins a ON a.id = al.admin_id
		ORDER BY al.created_at DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, fmt.Errorf("GetActivity: %w", err)
	}
	defer rows.Close()

	logs := []ActivityLog{}
	for rows.Next() {
		var l ActivityLog
		if err := rows.Scan(&l.ID, &l.AdminName, &l.Action, &l.Resource, &l.IP, &l.Timestamp); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// ── Private helpers ───────────────────────────────────────────────────────────

// sparklineCounts returns []int of daily row counts for the last N days.
// whereClause is appended as "AND <clause>" when non-empty — use only trusted/internal strings.
func (s *DashboardService) sparklineCounts(
	ctx context.Context, table, timeCol, whereClause string, days int,
) []int {
	extra := ""
	if whereClause != "" {
		extra = "AND " + whereClause
	}
	q := fmt.Sprintf(`
		SELECT COALESCE(COUNT(t.*), 0)::int
		FROM generate_series(
			CURRENT_DATE - INTERVAL '%d days',
			CURRENT_DATE,
			'1 day'::interval
		) AS d(day)
		LEFT JOIN %s t ON DATE(t.%s) = d.day %s
		GROUP BY d.day
		ORDER BY d.day
	`, days-1, table, timeCol, extra)

	return s.scanInts(ctx, q, days)
}

// sparklineSum returns []int of daily column sums for the last N days.
func (s *DashboardService) sparklineSum(
	ctx context.Context, table, sumCol, timeCol string, days int,
) []int {
	q := fmt.Sprintf(`
		SELECT COALESCE(SUM(t.%s), 0)::int
		FROM generate_series(
			CURRENT_DATE - INTERVAL '%d days',
			CURRENT_DATE,
			'1 day'::interval
		) AS d(day)
		LEFT JOIN %s t ON DATE(t.%s) = d.day
		GROUP BY d.day
		ORDER BY d.day
	`, sumCol, days-1, table, timeCol)

	return s.scanInts(ctx, q, days)
}

func (s *DashboardService) scanInts(ctx context.Context, q string, fallbackLen int) []int {
	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return make([]int, fallbackLen)
	}
	defer rows.Close()

	var result []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			result = append(result, 0)
			continue
		}
		result = append(result, v)
	}
	if len(result) == 0 {
		return make([]int, fallbackLen)
	}
	return result
}