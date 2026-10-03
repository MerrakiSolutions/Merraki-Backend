import { FastifyInstance } from 'fastify'
import { createOrderSchema } from './checkout.schema.js'
import { createCheckoutOrder } from './checkout.service.js'
import { getClientIp, rateLimit } from '../payments/payments.lib.js'

// prefix: /api/checkout
export const checkoutRoutes = async (app: FastifyInstance) => {
  app.post('/create-order', rateLimit(8, '1 minute'), async (request, reply) => {
    const body = createOrderSchema.parse(request.body)
    const data = await createCheckoutOrder(body, getClientIp(request))
    return reply.status(201).send({ success: true, data })
  })
}