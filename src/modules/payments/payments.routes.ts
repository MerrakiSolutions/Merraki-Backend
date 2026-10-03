import { FastifyInstance } from 'fastify'
import { z } from 'zod'
import { handleWebhook, settlePayment } from './payments.service.js'
import { getDownloadsByToken } from '../orders/orders.service.js'
import { AppError } from '../../lib/errors.js'
import { getUsdInr, rateLimit, verifyCheckoutSignature } from './payments.lib.js'

const verifySchema = z.object({
  razorpay_order_id: z.string().min(5).max(100),
  razorpay_payment_id: z.string().min(5).max(100),
  razorpay_signature: z.string().length(64),
})

// prefix: /api/payments
export const paymentsRoutes = async (app: FastifyInstance) => {
  app.get('/exchange-rate', rateLimit(60, '1 minute'), async (_req, reply) => {
    return reply.send({ success: true, data: { usdToInr: await getUsdInr(), fetchedAt: new Date() } })
  })

  // Browser calls this after Razorpay Checkout succeeds
  app.post('/verify', rateLimit(20, '1 minute'), async (request, reply) => {
    const b = verifySchema.parse(request.body)
    if (!verifyCheckoutSignature(b.razorpay_order_id, b.razorpay_payment_id, b.razorpay_signature)) {
      request.log.warn({ orderId: b.razorpay_order_id }, 'invalid payment signature on /verify')
      throw new AppError('Payment signature could not be verified.', 400, 'INVALID_SIGNATURE')
    }

    const { status, order } = await settlePayment(b.razorpay_order_id, b.razorpay_payment_id)

    if (status === 'paid') {
      const { orderId, guestName, items, expiresIn } = await getDownloadsByToken(order.downloadToken)
      return reply.send({
        success: true,
        data: {
          status: 'paid',
          orderId,
          downloadToken: order.downloadToken,
          guestName,
          items, // [{ title, downloadUrl }]
          expiresIn,
        },
      })
    }
    if (status === 'processing') {
      // not captured yet — the webhook finishes it; frontend polls /api/orders/status/:id
      return reply.status(202).send({ success: true, data: { status: 'processing', orderId: order.id } })
    }
    if (status === 'failed') {
      return reply.status(402).send({ success: false, error: { code: 'PAYMENT_FAILED', message: 'The payment did not go through. You have not been charged — please try again.' } })
    }
    throw new AppError('This order has been refunded.', 409, 'ORDER_REFUNDED')
  })

  // Razorpay → us. Authenticated by HMAC, so no rate limit.
  app.post('/webhook', { config: { rateLimit: false } }, async (request, reply) => {
    const sig = request.headers['x-razorpay-signature']
    await handleWebhook((request as any).rawBody as string | undefined, typeof sig === 'string' ? sig : undefined)
    return reply.status(200).send({ success: true })
  })
}