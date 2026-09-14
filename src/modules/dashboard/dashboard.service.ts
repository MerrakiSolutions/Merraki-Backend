import { eq, gte, desc, count, sum, and, sql } from 'drizzle-orm'
import { db } from '../../db/index.js'
import { orders } from '../../db/schema/orders.js'
import { templates } from '../../db/schema/templates.js'
import { blogPosts } from '../../db/schema/blog.js'
import { newsletterSubscribers } from '../../db/schema/newsletter.js'
import { contacts } from '../../db/schema/contacts.js'
import { founderTestResults } from '../../db/schema/founder-tests.js'
import { users, adminActivityLogs } from '../../db/schema/users.js'

// ── Helpers ───────────────────────────────────────────────────────

const startOf = (unit: 'day' | 'week' | 'month' | 'year'): Date => {
    const now = new Date()
    switch (unit) {
        case 'day':
            return new Date(now.getFullYear(), now.getMonth(), now.getDate())
        case 'week': {
            const day = now.getDay()
            const diff = now.getDate() - day + (day === 0 ? -6 : 1)
            return new Date(now.getFullYear(), now.getMonth(), diff)
        }
        case 'month':
            return new Date(now.getFullYear(), now.getMonth(), 1)
        case 'year':
            return new Date(now.getFullYear(), 0, 1)
    }
}

// ════════════════════════════════════════════════
// OVERVIEW STATS
// Top-level numbers shown on dashboard home
// ════════════════════════════════════════════════

export const getOverviewStats = async () => {
    const thisMonth = startOf('month')
    const thisYear = startOf('year')

    const [
        // orders
        totalOrdersResult,
        paidOrdersResult,
        monthOrdersResult,
        revenueResult,
        monthRevenueResult,

        // templates
        totalTemplatesResult,
        publishedTemplatesResult,

        // blog
        totalPostsResult,
        publishedPostsResult,

        // newsletter
        totalSubscribersResult,
        confirmedSubscribersResult,

        // contacts
        totalContactsResult,
        newContactsResult,

        // founder test
        totalLeadsResult,
        monthLeadsResult,

        // admins
        totalAdminsResult,
    ] = await Promise.all([
        // orders
        db.select({ total: count() }).from(orders),
        db.select({ total: count() }).from(orders).where(eq(orders.status, 'paid')),
        db.select({ total: count() }).from(orders).where(
            and(eq(orders.status, 'paid'), gte(orders.createdAt, thisMonth))
        ),

        // revenue — sum of totalUsd for paid orders
        db.select({ total: sum(orders.totalUsd) }).from(orders).where(eq(orders.status, 'paid')),
        db.select({ total: sum(orders.totalUsd) }).from(orders).where(
            and(eq(orders.status, 'paid'), gte(orders.createdAt, thisMonth))
        ),

        // templates
        db.select({ total: count() }).from(templates),
        db.select({ total: count() }).from(templates).where(eq(templates.status, 'published')),

        // blog
        db.select({ total: count() }).from(blogPosts),
        db.select({ total: count() }).from(blogPosts).where(eq(blogPosts.status, 'published')),

        // newsletter
        db.select({ total: count() }).from(newsletterSubscribers),
        db.select({ total: count() }).from(newsletterSubscribers).where(
            eq(newsletterSubscribers.confirmed, true)
        ),

        // contacts
        db.select({ total: count() }).from(contacts),
        db.select({ total: count() }).from(contacts).where(eq(contacts.status, 'new')),

        // founder test leads
        db.select({ total: count() }).from(founderTestResults),
        db.select({ total: count() }).from(founderTestResults).where(
            gte(founderTestResults.createdAt, thisMonth)
        ),

        // admins
        db.select({ total: count() }).from(users),
    ])

    return {
        orders: {
            total: Number(totalOrdersResult[0].total),
            paid: Number(paidOrdersResult[0].total),
            thisMonth: Number(monthOrdersResult[0].total),
        },
        revenue: {
            totalUsd: Number(revenueResult[0].total ?? 0).toFixed(2),
            thisMonthUsd: Number(monthRevenueResult[0].total ?? 0).toFixed(2),
        },
        templates: {
            total: Number(totalTemplatesResult[0].total),
            published: Number(publishedTemplatesResult[0].total),
        },
        blog: {
            total: Number(totalPostsResult[0].total),
            published: Number(publishedPostsResult[0].total),
        },
        newsletter: {
            total: Number(totalSubscribersResult[0].total),
            confirmed: Number(confirmedSubscribersResult[0].total),
        },
        contacts: {
            total: Number(totalContactsResult[0].total),
            new: Number(newContactsResult[0].total),
        },
        leads: {
            total: Number(totalLeadsResult[0].total),
            thisMonth: Number(monthLeadsResult[0].total),
        },
        admins: {
            total: Number(totalAdminsResult[0].total),
        },
    }
}

// ════════════════════════════════════════════════
// REVENUE CHART
// Daily revenue for last N days
// ════════════════════════════════════════════════

export const getRevenueChart = async (days = 30) => {
    const since = new Date()
    since.setDate(since.getDate() - days)

    const data = await db
        .select({
            date: sql<string>`DATE(${orders.createdAt})`.as('date'),
            revenue: sum(orders.totalUsd).as('revenue'),
            orderCount: count().as('order_count'),
        })
        .from(orders)
        .where(
            and(
                eq(orders.status, 'paid'),
                gte(orders.createdAt, since)
            )
        )
        .groupBy(sql`DATE(${orders.createdAt})`)
        .orderBy(sql`DATE(${orders.createdAt})`)

    // fill in missing days with zero
    const result: { date: string; revenue: string; orderCount: number }[] = []
    const dateMap = new Map(data.map((d) => [d.date, d]))

    for (let i = days - 1; i >= 0; i--) {
        const d = new Date()
        d.setDate(d.getDate() - i)
        const dateStr = d.toISOString().split('T')[0]
        const entry = dateMap.get(dateStr)

        result.push({
            date: dateStr,
            revenue: Number(entry?.revenue ?? 0).toFixed(2),
            orderCount: Number(entry?.orderCount ?? 0),
        })
    }

    return result
}

// ════════════════════════════════════════════════
// TOP TEMPLATES
// Best selling templates by order count
// ════════════════════════════════════════════════

export const getTopTemplates = async (limit = 5) => {
    // orders.items is JSONB array of { templateId, title, priceUsd, r2Key }
    // we unnest and group by templateId to get counts
    const data = await db.execute(sql`
    SELECT
      item->>'templateId' AS template_id,
      item->>'title'      AS title,
      item->>'priceUsd'   AS price_usd,
      COUNT(*)::int       AS order_count,
      SUM((item->>'priceUsd')::numeric)::numeric AS total_revenue
    FROM ${orders},
    jsonb_array_elements(${orders.items}) AS item
    WHERE ${orders.status} = 'paid'
    GROUP BY item->>'templateId', item->>'title', item->>'priceUsd'
    ORDER BY order_count DESC
    LIMIT ${limit}
  `)

    return (data as any[]).map((row) => ({
        templateId: row.template_id,
        title: row.title,
        priceUsd: row.price_usd,
        orderCount: Number(row.order_count),
        totalRevenue: Number(row.total_revenue).toFixed(2),
    }))
}

// ════════════════════════════════════════════════
// RECENT ORDERS
// Latest N paid orders
// ════════════════════════════════════════════════

export const getRecentOrders = async (limit = 10) => {
    const data = await db
        .select({
            id: orders.id,
            guestName: orders.guestName,
            guestEmail: orders.guestEmail,
            totalUsd: orders.totalUsd,
            currencyCharged: orders.currencyCharged,
            amountCharged: orders.amountCharged,
            status: orders.status,
            createdAt: orders.createdAt,
        })
        .from(orders)
        .orderBy(desc(orders.createdAt))
        .limit(limit)

    return data
}

// ════════════════════════════════════════════════
// RECENT CONTACTS
// Latest N unread contacts
// ════════════════════════════════════════════════

export const getRecentContacts = async (limit = 5) => {
    return db
        .select({
            id: contacts.id,
            name: contacts.name,
            email: contacts.email,
            message: contacts.message,
            status: contacts.status,
            createdAt: contacts.createdAt,
        })
        .from(contacts)
        .where(eq(contacts.status, 'new'))
        .orderBy(desc(contacts.createdAt))
        .limit(limit)
}

// ════════════════════════════════════════════════
// RECENT LEADS
// Latest N founder test submissions
// ════════════════════════════════════════════════

export const getRecentLeads = async (limit = 5) => {
    return db
        .select({
            id: founderTestResults.id,
            leadName: founderTestResults.leadName,
            leadEmail: founderTestResults.leadEmail,
            leadCompany: founderTestResults.leadCompany,
            resultType: founderTestResults.resultType,
            score: founderTestResults.score,
            createdAt: founderTestResults.createdAt,
        })
        .from(founderTestResults)
        .orderBy(desc(founderTestResults.createdAt))
        .limit(limit)
}

// ════════════════════════════════════════════════
// FOUNDER TEST BREAKDOWN
// Result type distribution
// ════════════════════════════════════════════════

export const getLeadBreakdown = async () => {
    const data = await db
        .select({
            resultType: founderTestResults.resultType,
            total: count(),
        })
        .from(founderTestResults)
        .groupBy(founderTestResults.resultType)
        .orderBy(desc(count()))

    const totalLeads = data.reduce((sum, r) => sum + Number(r.total), 0)

    return data.map((r) => ({
        resultType: r.resultType ?? 'Unknown',
        count: Number(r.total),
        percentage: totalLeads > 0
            ? Math.round((Number(r.total) / totalLeads) * 100)
            : 0,
    }))
}

// ════════════════════════════════════════════════
// NEWSLETTER GROWTH
// Subscriber signups per day for last N days
// ════════════════════════════════════════════════

export const getNewsletterGrowth = async (days = 30) => {
    const since = new Date()
    since.setDate(since.getDate() - days)

    const data = await db
        .select({
            date: sql<string>`DATE(${newsletterSubscribers.createdAt})`.as('date'),
            signups: count().as('signups'),
        })
        .from(newsletterSubscribers)
        .where(
            and(
                eq(newsletterSubscribers.confirmed, true),
                gte(newsletterSubscribers.createdAt, since)
            )
        )
        .groupBy(sql`DATE(${newsletterSubscribers.createdAt})`)
        .orderBy(sql`DATE(${newsletterSubscribers.createdAt})`)

    // fill in missing days
    const result: { date: string; signups: number }[] = []
    const dateMap = new Map(data.map((d) => [d.date, d]))

    for (let i = days - 1; i >= 0; i--) {
        const d = new Date()
        d.setDate(d.getDate() - i)
        const dateStr = d.toISOString().split('T')[0]
        const entry = dateMap.get(dateStr)
        result.push({ date: dateStr, signups: Number(entry?.signups ?? 0) })
    }

    return result
}

// ════════════════════════════════════════════════
// RECENT ADMIN ACTIVITY
// Last N actions across all admins
// ════════════════════════════════════════════════

export const getRecentAdminActivity = async (limit = 10) => {
    return db
        .select({
            id: adminActivityLogs.id,
            userId: adminActivityLogs.userId,
            adminName: users.name,
            adminEmail: users.email,
            action: adminActivityLogs.action,
            ipAddress: adminActivityLogs.ipAddress,
            createdAt: adminActivityLogs.createdAt,
        })
        .from(adminActivityLogs)
        .leftJoin(users, eq(adminActivityLogs.userId, users.id))
        .orderBy(desc(adminActivityLogs.createdAt))
        .limit(limit)
}

// ════════════════════════════════════════════════
// FULL DASHBOARD
// One call returns everything the dashboard needs
// ════════════════════════════════════════════════

export const getFullDashboard = async () => {
    const [
        overview,
        revenueChart,
        topTemplates,
        recentOrders,
        recentContacts,
        recentLeads,
        leadBreakdown,
        newsletterGrowth,
        recentActivity,
    ] = await Promise.all([
        getOverviewStats(),
        getRevenueChart(30),
        getTopTemplates(5),
        getRecentOrders(10),
        getRecentContacts(5),
        getRecentLeads(5),
        getLeadBreakdown(),
        getNewsletterGrowth(30),
        getRecentAdminActivity(10),
    ])

    return {
        overview,
        revenueChart,
        topTemplates,
        recentOrders,
        recentContacts,
        recentLeads,
        leadBreakdown,
        newsletterGrowth,
        recentActivity,
    }
}