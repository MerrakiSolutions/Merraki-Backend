import { FastifyInstance } from 'fastify'
import * as svc from './orders.service.js'
import { fulfillOrder } from './orders.email.js'
import { authenticate } from '../../middleware/authenticate.js'
import { requireAdmin } from '../../middleware/require-admin.js'
import { AppError } from '../../lib/errors.js'
import { rateLimit } from '../payments/payments.lib.js'
import { adminOrdersQuerySchema, idParamSchema, tokenParamSchema, trackQuerySchema, trackRequestSchema } from './orders.schema.js'

// prefix: /api/orders
export const publicOrderRoutes = async (app: FastifyInstance) => {
  // By order id only. Email lookup moved to POST /track/request (the old version exposed download tokens).
  app.get('/track', rateLimit(30, '1 minute'), async (request, reply) => {
    const q = trackQuerySchema.parse(request.query)
    if (q.email) throw new AppError('Email lookup now sends a secure link. Use POST /api/orders/track/request.', 400, 'USE_TRACK_REQUEST')
    return reply.send({ success: true, data: q.order_id ? await svc.trackOrderById(q.order_id) : [] })
  })

  app.post('/track/request', rateLimit(5, '1 hour'), async (request, reply) => {
    const { email } = trackRequestSchema.parse(request.body)
    await svc.requestOrderLinks(email)
    // identical response whether or not orders exist — no email enumeration
    return reply.status(202).send({ success: true, message: 'If orders exist for that email, we have sent you a secure link.' })
  })

  app.get('/status/:id', rateLimit(60, '1 minute'), async (request, reply) => {
    const { id } = idParamSchema.parse(request.params)
    return reply.send({ success: true, data: await svc.getOrderStatus(id) })
  })

  app.get('/download/:token', rateLimit(20, '1 minute'), async (request, reply) => {
    const { token } = tokenParamSchema.parse(request.params)
    return reply.send({ success: true, data: await svc.getDownloadsByToken(token) })
  })
}

// prefix: /api/admin/orders
export const adminOrderRoutes = async (app: FastifyInstance) => {
  app.addHook('preHandler', authenticate)
  app.addHook('preHandler', requireAdmin)

  app.get('/', async (request, reply) => {
    return reply.send({ success: true, ...(await svc.listAdminOrders(adminOrdersQuerySchema.parse(request.query))) })
  })

  app.get('/export', async (_request, reply) => {
    return reply
      .header('Content-Type', 'text/csv')
      .header('Content-Disposition', 'attachment; filename="orders.csv"')
      .send(await svc.exportOrdersCsv())
  })

  app.get('/:id', async (request, reply) => {
    const { id } = idParamSchema.parse(request.params)
    return reply.send({ success: true, data: await svc.getAdminOrder(id) })
  })

  app.post('/:id/resend-email', async (request, reply) => {
    const { id } = idParamSchema.parse(request.params)
    const order = await svc.getAdminOrder(id)
    if (order.status !== 'paid') throw new AppError('Only paid orders can be re-sent.', 409)
    await fulfillOrder(order, { force: true })
    return reply.send({ success: true, message: 'Receipt sent.' })
  })

  app.delete('/:id', async (request, reply) => {
    const { id } = idParamSchema.parse(request.params)
    return reply.send({ success: true, ...(await svc.deleteOrder(id)) })
  })
}