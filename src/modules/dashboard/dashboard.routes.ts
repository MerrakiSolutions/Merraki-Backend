import { FastifyInstance } from 'fastify'
import * as dashboardService from './dashboard.service.js'
import { authenticate } from '../../middleware/authenticate.js'
import { requireAdmin } from '../../middleware/require-admin.js'
import { z } from 'zod'

const chartQuerySchema = z.object({
    days: z.coerce.number().int().min(7).max(365).default(30),
})

const limitQuerySchema = z.object({
    limit: z.coerce.number().int().min(1).max(100).default(10),
})

export const dashboardRoutes = async (app: FastifyInstance) => {

    app.addHook('preHandler', authenticate)
    app.addHook('preHandler', requireAdmin)

    // ── GET /api/admin/dashboard ─────────────────────
    // Full dashboard in one call — frontend loads this on mount
    app.get('/', async (_request, reply) => {
        const data = await dashboardService.getFullDashboard()
        return reply.send({ success: true, data })
    })

    // ── GET /api/admin/dashboard/overview ────────────
    // Top level counts only — lightweight refresh
    app.get('/overview', async (_request, reply) => {
        const data = await dashboardService.getOverviewStats()
        return reply.send({ success: true, data })
    })

    // ── GET /api/admin/dashboard/revenue ─────────────
    // Revenue chart data — ?days=30
    app.get('/revenue', async (request, reply) => {
        const { days } = chartQuerySchema.parse(request.query)
        const data = await dashboardService.getRevenueChart(days)
        return reply.send({ success: true, data })
    })

    // ── GET /api/admin/dashboard/top-templates ───────
    // Best selling templates — ?limit=5
    app.get('/top-templates', async (request, reply) => {
        const { limit } = limitQuerySchema.parse(request.query)
        const data = await dashboardService.getTopTemplates(limit)
        return reply.send({ success: true, data })
    })

    // ── GET /api/admin/dashboard/recent-orders ───────
    app.get('/recent-orders', async (request, reply) => {
        const { limit } = limitQuerySchema.parse(request.query)
        const data = await dashboardService.getRecentOrders(limit)
        return reply.send({ success: true, data })
    })

    // ── GET /api/admin/dashboard/recent-contacts ─────
    app.get('/recent-contacts', async (request, reply) => {
        const { limit } = limitQuerySchema.parse(request.query)
        const data = await dashboardService.getRecentContacts(limit)
        return reply.send({ success: true, data })
    })

    // ── GET /api/admin/dashboard/recent-leads ────────
    app.get('/recent-leads', async (request, reply) => {
        const { limit } = limitQuerySchema.parse(request.query)
        const data = await dashboardService.getRecentLeads(limit)
        return reply.send({ success: true, data })
    })

    // ── GET /api/admin/dashboard/lead-breakdown ──────
    app.get('/lead-breakdown', async (_request, reply) => {
        const data = await dashboardService.getLeadBreakdown()
        return reply.send({ success: true, data })
    })

    // ── GET /api/admin/dashboard/newsletter-growth ───
    app.get('/newsletter-growth', async (request, reply) => {
        const { days } = chartQuerySchema.parse(request.query)
        const data = await dashboardService.getNewsletterGrowth(days)
        return reply.send({ success: true, data })
    })

    // ── GET /api/admin/dashboard/activity ────────────
    app.get('/activity', async (request, reply) => {
        const { limit } = limitQuerySchema.parse(request.query)
        const data = await dashboardService.getRecentAdminActivity(limit)
        return reply.send({ success: true, data })
    })
}