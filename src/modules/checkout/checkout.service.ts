import { randomUUID } from 'node:crypto'
import { eq, inArray } from 'drizzle-orm'
import { db } from '../../db/index.js'
import { orders, type OrderItem } from '../../db/schema/orders.js'
import { templates } from '../../db/schema/templates.js'
import { AppError } from '../../lib/errors.js'
import { env } from '../../config/env.js'
import {
  razorpay, isLiveMode, describeRazorpayError, toMinor, fromMinor, getUsdInr, sanitizeText, looksLikeLink,
} from '../payments/payments.lib.js'
import type { CreateOrderInput } from './checkout.schema.js'

// Cards in USD, UPI always INR. Set CARD_CURRENCY=INR in .env if USD isn't enabled on your Razorpay account.
const CARD_CURRENCY = process.env.CARD_CURRENCY === 'INR' ? 'INR' : 'USD'

export async function createCheckoutOrder(input: CreateOrderInput, ip: string | null) {
  // 1. sanitise — these strings end up in emails we send
  const name = sanitizeText(input.guestName, 100)
  const a = input.billingAddress
  const company = a.company ? sanitizeText(a.company, 120) : undefined
  if (name.length < 2 || looksLikeLink(name) || (company && looksLikeLink(company))) {
    throw new AppError('Please enter your real name — links and email addresses are not allowed.', 400, 'INVALID_NAME')
  }
  const email = input.guestEmail.trim().toLowerCase()
  const billingAddress = {
    line1: sanitizeText(a.line1, 150),
    line2: a.line2 ? sanitizeText(a.line2, 150) : undefined,
    city: sanitizeText(a.city, 80),
    state: sanitizeText(a.state, 80),
    country: sanitizeText(a.country, 60),
    zip: sanitizeText(a.zip, 12),
    company,
  }

  // 2. prices come from the DB — never from the client
  const ids = [...new Set(input.items.map((i) => i.templateId))]
  const found = await db
    .select({ id: templates.id, title: templates.title, priceUsd: templates.priceUsd, r2Key: templates.r2Key, status: templates.status })
    .from(templates)
    .where(inArray(templates.id, ids))
  const byId = new Map(found.map((t) => [t.id, t]))
  if (ids.some((id) => byId.get(id)?.status !== 'published' || !byId.get(id)?.r2Key)) {
    throw new AppError('One or more items are no longer available.', 400, 'ITEM_UNAVAILABLE')
  }

  const items: OrderItem[] = ids.map((id) => {
    const t = byId.get(id)!
    return { templateId: id, title: t.title, priceUsd: fromMinor(toMinor(t.priceUsd)), r2Key: t.r2Key }
  })
  const totalCents = items.reduce((sum, i) => sum + toMinor(i.priceUsd), 0)
  if (totalCents <= 0) throw new AppError('This order cannot be paid for online.', 400)

  // 3. charge currency + amount in the smallest unit
  const currency = input.paymentMethod === 'upi' ? 'INR' : CARD_CURRENCY
  const rate = currency === 'INR' ? await getUsdInr() : null
  const amountMinor = rate ? Math.round(totalCents * rate) : totalCents // USD cents × rate = INR paise

  // 4. our order first, then Razorpay's
  const orderId = randomUUID()
  await db.insert(orders).values({
    id: orderId,
    guestName: name,
    guestEmail: email,
    billingAddress,
    items,
    subtotalUsd: fromMinor(totalCents),
    totalUsd: fromMinor(totalCents),
    currencyCharged: currency,
    exchangeRate: rate ? rate.toFixed(4) : null,
    amountCharged: fromMinor(amountMinor),
    ipAddress: ip,
    livemode: isLiveMode,
  })

  try {
    const rzp = await razorpay.orders.create({ amount: amountMinor, currency, receipt: orderId, notes: { orderId } })
    await db.update(orders).set({ razorpayOrderId: rzp.id, updatedAt: new Date() }).where(eq(orders.id, orderId))
    return {
      orderId,
      razorpayOrderId: rzp.id,
      keyId: env.RAZORPAY_KEY_ID,
      amount: amountMinor,
      currency,
      totalUsd: fromMinor(totalCents),
      exchangeRate: rate ? rate.toFixed(4) : null,
      name: 'MerrakiSolutions',
      description: `${items.length} item${items.length > 1 ? 's' : ''}`,
      prefill: { name, email },
    }
  } catch (err) {
    console.error('Razorpay order creation failed', orderId, describeRazorpayError(err))
    await db.update(orders).set({ status: 'failed', updatedAt: new Date() }).where(eq(orders.id, orderId))
    throw new AppError('The payment service is temporarily unavailable. Please try again in a moment.', 502, 'PAYMENT_PROVIDER_UNAVAILABLE')
  }
}