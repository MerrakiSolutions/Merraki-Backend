import { FastifyInstance } from 'fastify'
import * as founderTestsService from './founder-tests.service.js'
import { authenticate } from '../../middleware/authenticate.js'
import { requireAdmin } from '../../middleware/require-admin.js'
import {
    submitTestSchema,
    founderTestQuerySchema,
    createQuestionSchema,
    updateQuestionSchema,
    reorderQuestionsSchema,
    createArchetypeSchema,
    updateArchetypeSchema,
} from './founder-tests.schema.js'

// ════════════════════════════════════════════════
// PUBLIC ROUTES — prefix: /api/founder-test
// ════════════════════════════════════════════════

export const publicFounderTestRoutes = async (app: FastifyInstance) => {
    // GET /api/founder-test/questions
    app.get('/questions', async (_request, reply) => {
        const data = await founderTestsService.getPublicQuestions()
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
// ADMIN ROUTES — prefix: /api/admin/founder-test
// ════════════════════════════════════════════════

export const adminFounderTestRoutes = async (app: FastifyInstance) => {
    app.addHook('preHandler', authenticate)
    app.addHook('preHandler', requireAdmin)

    // ── Leads (existing) ──────────────────────────────────────────────

    app.get('/', async (request, reply) => {
        const query = founderTestQuerySchema.parse(request.query)
        const result = await founderTestsService.getAdminResults(query)
        return reply.send({ success: true, ...result })
    })

    app.get('/stats', async (_request, reply) => {
        const data = await founderTestsService.getResultStats()
        return reply.send({ success: true, data })
    })

    app.get('/export', async (_request, reply) => {
        const csv = await founderTestsService.exportResultsCSV()
        return reply
            .header('Content-Type', 'text/csv')
            .header('Content-Disposition', 'attachment; filename="founder-test-results.csv"')
            .send(csv)
    })

    app.get('/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const data = await founderTestsService.getAdminResultById(id)
        return reply.send({ success: true, data })
    })

    app.delete('/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const result = await founderTestsService.deleteResult(id)
        return reply.send({ success: true, ...result })
    })

    // ── Questions CRUD ────────────────────────────────────────────────

    app.get('/questions', async (_request, reply) => {
        const data = await founderTestsService.getAdminQuestions()
        return reply.send({ success: true, data })
    })

    app.get('/questions/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const data = await founderTestsService.getAdminQuestionById(id)
        return reply.send({ success: true, data })
    })

    app.post('/questions', async (request, reply) => {
        const body = createQuestionSchema.parse(request.body)
        const data = await founderTestsService.createQuestion(body)
        return reply.status(201).send({ success: true, data })
    })

    app.put('/questions/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const body = updateQuestionSchema.parse(request.body)
        const data = await founderTestsService.updateQuestion(id, body)
        return reply.send({ success: true, data })
    })

    app.delete('/questions/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const result = await founderTestsService.deleteQuestion(id)
        return reply.send({ success: true, ...result })
    })

    app.put('/questions/reorder', async (request, reply) => {
        const body = reorderQuestionsSchema.parse(request.body)
        const result = await founderTestsService.reorderQuestions(body.items)
        return reply.send({ success: true, ...result })
    })

    // ── Archetypes CRUD ───────────────────────────────────────────────

    app.get('/archetypes', async (_request, reply) => {
        const data = await founderTestsService.getAdminArchetypes()
        return reply.send({ success: true, data })
    })

    app.get('/archetypes/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const data = await founderTestsService.getAdminArchetypeById(id)
        return reply.send({ success: true, data })
    })

    app.post('/archetypes', async (request, reply) => {
        const body = createArchetypeSchema.parse(request.body)
        const data = await founderTestsService.createArchetype(body)
        return reply.status(201).send({ success: true, data })
    })

    app.put('/archetypes/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const body = updateArchetypeSchema.parse(request.body)
        const data = await founderTestsService.updateArchetype(id, body)
        return reply.send({ success: true, data })
    })

    app.delete('/archetypes/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const result = await founderTestsService.deleteArchetype(id)
        return reply.send({ success: true, ...result })
    })
}