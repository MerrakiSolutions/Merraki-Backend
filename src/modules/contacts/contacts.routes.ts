import { FastifyInstance } from 'fastify'
import * as contactsService from './contacts.service.js'
import { authenticate } from '../../middleware/authenticate.js'
import { requireAdmin } from '../../middleware/require-admin.js'
import {
    createContactSchema,
    updateContactStatusSchema,
    contactQuerySchema,
} from './contacts.schema.js'

// ════════════════════════════════════════════════
// PUBLIC ROUTES
// prefix: /api/contacts
// ════════════════════════════════════════════════

export const publicContactRoutes = async (app: FastifyInstance) => {

    // POST /api/contacts
    app.post('/', async (request, reply) => {
        const body = createContactSchema.parse(request.body)
        const result = await contactsService.submitContact({
            ...body,
            ipAddress: request.ip,
        })
        return reply.status(201).send({ success: true, ...result })
    })
}

// ════════════════════════════════════════════════
// ADMIN ROUTES
// prefix: /api/admin/contacts
// ════════════════════════════════════════════════

export const adminContactRoutes = async (app: FastifyInstance) => {

    app.addHook('preHandler', authenticate)
    app.addHook('preHandler', requireAdmin)

    // GET /api/admin/contacts
    app.get('/', async (request, reply) => {
        const query = contactQuerySchema.parse(request.query)
        const result = await contactsService.getAdminContacts(query)
        return reply.send({ success: true, ...result })
    })

    // GET /api/admin/contacts/export
    app.get('/export', async (_request, reply) => {
        const csv = await contactsService.exportContactsCSV()
        return reply
            .header('Content-Type', 'text/csv')
            .header('Content-Disposition', 'attachment; filename="contacts.csv"')
            .send(csv)
    })

    // GET /api/admin/contacts/:id
    app.get('/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const data = await contactsService.getAdminContactById(id)
        return reply.send({ success: true, data })
    })

    // PUT /api/admin/contacts/:id/status
    app.put('/:id/status', async (request, reply) => {
        const { id } = request.params as { id: string }
        const body = updateContactStatusSchema.parse(request.body)
        const data = await contactsService.updateContactStatus(id, body.status)
        return reply.send({ success: true, data })
    })

    // DELETE /api/admin/contacts/:id
    app.delete('/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const result = await contactsService.deleteContact(id)
        return reply.send({ success: true, ...result })
    })
}