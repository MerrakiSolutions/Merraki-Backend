import { FastifyInstance } from 'fastify'
import * as founderTestsService from './founder-tests.service.js'
import { authenticate } from '../../middleware/authenticate.js'
import { requireAdmin } from '../../middleware/require-admin.js'
import {
    submitTestSchema,
    founderTestQuerySchema,
} from './founder-tests.schema.js'

// ════════════════════════════════════════════════
// PUBLIC ROUTES
// prefix: /api/founder-test
// ════════════════════════════════════════════════

export const publicFounderTestRoutes = async (app: FastifyInstance) => {

    // GET /api/founder-test/questions
    // Returns questions without scores — safe for public
    app.get('/questions', async (_request, reply) => {
        const data = founderTestsService.getTestQuestions()
        return reply.send({ success: true, data })
    })

    // GET /api/founder-test/result-types
    // All possible archetypes — use on landing page
    app.get('/result-types', async (_request, reply) => {
        const data = founderTestsService.getResultTypes()
        return reply.send({ success: true, data })
    })

    // POST /api/founder-test/submit
    app.post('/submit', async (request, reply) => {
        const body = submitTestSchema.parse(request.body)
        const result = await founderTestsService.submitTest({
            ...body,
            ipAddress: request.ip,
        })
        return reply.status(201).send({ success: true, data: result })
    })
}

// ════════════════════════════════════════════════
// ADMIN ROUTES
// prefix: /api/admin/founder-test
// ════════════════════════════════════════════════

export const adminFounderTestRoutes = async (app: FastifyInstance) => {

    app.addHook('preHandler', authenticate)
    app.addHook('preHandler', requireAdmin)

    // GET /api/admin/founder-test
    app.get('/', async (request, reply) => {
        const query = founderTestQuerySchema.parse(request.query)
        const result = await founderTestsService.getAdminResults(query)
        return reply.send({ success: true, ...result })
    })

    // GET /api/admin/founder-test/stats
    // Breakdown by result type — useful for dashboard
    app.get('/stats', async (_request, reply) => {
        const data = await founderTestsService.getResultStats()
        return reply.send({ success: true, data })
    })

    // GET /api/admin/founder-test/export
    app.get('/export', async (_request, reply) => {
        const csv = await founderTestsService.exportResultsCSV()
        return reply
            .header('Content-Type', 'text/csv')
            .header('Content-Disposition', 'attachment; filename="founder-test-results.csv"')
            .send(csv)
    })

    // GET /api/admin/founder-test/:id
    app.get('/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const data = await founderTestsService.getAdminResultById(id)
        return reply.send({ success: true, data })
    })

    // DELETE /api/admin/founder-test/:id
    app.delete('/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const result = await founderTestsService.deleteResult(id)
        return reply.send({ success: true, ...result })
    })
}