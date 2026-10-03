import { and, eq, inArray } from 'drizzle-orm'
import { db } from '../../db/index.js'
import { orders, type Order } from '../../db/schema/orders.js'
import { AppError, NotFoundError } from '../../lib/errors.js'
import { fulfillOrder } from '../orders/orders.email.js'
import { alertAdmin, razorpay, toMinor, verifyWebhookSignature } from './payments.lib.js'

type SettleStatus = 'paid' | 'processing' | 'failed' | 'refunded'

async function fetchPayment(paymentId: string): Promise<any> {
  try {
    return await razorpay.payments.fetch(paymentId)
  } catch (err) {
    const code = (err as any)?.statusCode
    if (code === 400 || code === 404) throw new AppError('Unknown payment reference.', 400, 'UNKNOWN_PAYMENT')
    throw new AppError('Could not confirm the payment with the provider. Please retry shortly.', 502)
  }
}

export async function settlePayment(
  razorpayOrderId: string,
  paymentId: string
): Promise<{ status: SettleStatus; order: Order }> {
  const [order] = await db.select().from(orders).where(eq(orders.razorpayOrderId, razorpayOrderId)).limit(1)
  if (!order) throw new NotFoundError('Order not found.')

  // Never trust the request body: ask Razorpay what really happened.
  let payment = await fetchPayment(paymentId)
  if (payment.order_id !== razorpayOrderId) {
    throw new AppError('Payment does not belong to this order.', 400, 'PAYMENT_ORDER_MISMATCH')
  }
  if (payment.status === 'authorized') {
    try {
      payment = await razorpay.payments.capture(payment.id, Number(payment.amount), payment.currency)
    } catch {
      payment = await fetchPayment(payment.id) // probably auto-captured meanwhile
    }
  }
  if (payment.status === 'failed') return { status: 'failed', order }
  if (payment.status !== 'captured') return { status: 'processing', order }

  // Amount + currency must be exactly what WE priced.
  if (Number(payment.amount) !== toMinor(order.amountCharged) || payment.currency !== order.currencyCharged) {
    await alertAdmin('Payment amount mismatch — order NOT fulfilled', [
      `Order ${order.id}`, `Payment ${payment.id}`,
      `Expected ${order.amountCharged} ${order.currencyCharged}`, `Got ${payment.amount} (minor units) ${payment.currency}`,
    ])
    throw new AppError('Payment amount mismatch.', 409, 'AMOUNT_MISMATCH')
  }

  // Atomic: only ONE caller (verify or webhook, first or replayed) wins this update.
  const [paid] = await db
    .update(orders)
    .set({ status: 'paid', razorpayPaymentId: payment.id, paymentMethod: payment.method ?? null, paidAt: new Date(), updatedAt: new Date() })
    .where(and(eq(orders.id, order.id), inArray(orders.status, ['pending', 'failed']))) // a late capture of an old order still counts
    .returning()

  if (paid) {
    // Receipt email; if it fails the 5-minute sweep retries it.
    void fulfillOrder(paid).catch((err) => console.error('receipt send failed, sweep will retry', paid.id, err))
    return { status: 'paid', order: paid }
  }

  const [fresh] = await db.select().from(orders).where(eq(orders.id, order.id)).limit(1)
  if (fresh.razorpayPaymentId && fresh.razorpayPaymentId !== payment.id) {
    await alertAdmin('Duplicate payment — refund the extra one in Razorpay', [
      `Order ${order.id}`, `Keeping payment ${fresh.razorpayPaymentId}`, `Extra payment ${payment.id}`,
    ])
  }
  return { status: fresh.status === 'refunded' ? 'refunded' : 'paid', order: fresh }
}

export async function handleWebhook(rawBody: string | undefined, signature: string | undefined) {
  if (!rawBody || !signature) throw new AppError('Malformed webhook request.', 400)
  if (!verifyWebhookSignature(rawBody, signature)) {
    throw new AppError('Invalid webhook signature.', 401, 'INVALID_WEBHOOK_SIGNATURE')
  }

  let event: any
  try {
    event = JSON.parse(rawBody)
  } catch {
    throw new AppError('Malformed webhook body.', 400)
  }

  const pay = event?.payload?.payment?.entity

  switch (event?.event) {
    case 'payment.authorized':
    case 'payment.captured':
    case 'order.paid': {
      const razorpayOrderId = pay?.order_id ?? event?.payload?.order?.entity?.id
      if (!pay?.id || !razorpayOrderId) return
      try {
        await settlePayment(razorpayOrderId, pay.id) // idempotent — duplicate deliveries are harmless
      } catch (err) {
        // Business errors (unknown order, mismatch…) won't be fixed by a retry — acknowledge.
        // Infra errors (5xx / DB down) rethrow so Razorpay retries.
        if (err instanceof AppError && err.statusCode < 500) return console.warn('webhook ignored:', err.message)
        throw err
      }
      return
    }

    case 'refund.processed': {
      const refund = event?.payload?.refund?.entity
      if (!refund?.payment_id) return
      const [order] = await db.select().from(orders).where(eq(orders.razorpayPaymentId, refund.payment_id)).limit(1)
      if (!order) return
      if (Number(refund.amount) !== toMinor(order.amountCharged)) {
        await alertAdmin('Partial refund seen in Razorpay', [`Order ${order.id}`, `Refund ${refund.id}`, 'Orders only track full refunds — review manually.'])
        return
      }
      await db.update(orders).set({ status: 'refunded', updatedAt: new Date() }).where(and(eq(orders.id, order.id), eq(orders.status, 'paid')))
      return
    }

    default:
      return // payment.failed etc. — nothing to do; the buyer can simply retry
  }
}