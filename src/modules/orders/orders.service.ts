import { and, asc, count, desc, eq, ilike, inArray, or, sql } from 'drizzle-orm'
import type { z } from 'zod'
import { db } from '../../db/index.js'
import { orders, type Order } from '../../db/schema/orders.js'
import { templates } from '../../db/schema/templates.js'
import { AppError, NotFoundError } from '../../lib/errors.js'
import { paginate, getPaginationOffset } from '../../lib/pagination.js'
import { csvCell, presignDownload, UUID_RE } from '../payments/payments.lib.js'
import { sendOrderLinks } from './orders.email.js'
import type { adminOrdersQuerySchema } from './orders.schema.js'

// ── public ───────────────────────────────────────────────────────────

/** Never leaks r2 keys, payment ids, IPs or billing addresses. */
export function toPublicOrder(o: Order) {
  return {
    id: o.id,
    guestName: o.guestName,
    guestEmail: o.guestEmail,
    items: o.items.map(({ templateId, title, priceUsd }) => ({ templateId, title, priceUsd })),
    totalUsd: o.totalUsd,
    currencyCharged: o.currencyCharged,
    amountCharged: o.amountCharged,
    exchangeRate: o.exchangeRate,
    status: o.status,
    downloadToken: o.status === 'paid' ? o.downloadToken : null,
    createdAt: o.createdAt,
    paidAt: o.paidAt,
  }
}

/** The order id is an unguessable UUID — holding it is the buyer's proof of ownership. */
export async function trackOrderById(orderId: string) {
  if (!UUID_RE.test(orderId)) return []
  const [o] = await db.select().from(orders).where(eq(orders.id, orderId)).limit(1)
  return o ? [toPublicOrder(o)] : []
}

export async function getOrderStatus(orderId: string) {
  if (!UUID_RE.test(orderId)) throw new NotFoundError('Order not found.')
  const [o] = await db.select().from(orders).where(eq(orders.id, orderId)).limit(1)
  if (!o) throw new NotFoundError('Order not found.')
  return { status: o.status, downloadToken: o.status === 'paid' ? o.downloadToken : null }
}

// Email lookup never returns data — it emails links to the address owner. Max 1 email/hour/address.
const lastLookup = new Map<string, number>()
export async function requestOrderLinks(rawEmail: string) {
  const email = rawEmail.trim().toLowerCase()
  const now = Date.now()
  if ((lastLookup.get(email) ?? 0) > now - 3_600_000) return
  const rows = await db
    .select()
    .from(orders)
    .where(and(eq(sql`lower(${orders.guestEmail})`, email), eq(orders.status, 'paid')))
    .orderBy(desc(orders.createdAt))
    .limit(10)
  if (!rows.length) return
  if (lastLookup.size > 5000) lastLookup.clear()
  lastLookup.set(email, now)
  void sendOrderLinks(email, rows).catch((err) => console.error('order links email failed', err))
}

function downloadName(title: string, key: string) {
  const ext = key.includes('.') ? key.slice(key.lastIndexOf('.')).replace(/[^.\w]/g, '') : ''
  return `${title.replace(/[^\w\- ]+/g, '').trim().replace(/\s+/g, '-').slice(0, 80) || 'download'}${ext}`
}

export async function getDownloadsByToken(token: string) {
  if (!UUID_RE.test(token)) throw new NotFoundError('Download link not found.')
  const [order] = await db.select().from(orders).where(eq(orders.downloadToken, token)).limit(1)
  if (!order) throw new NotFoundError('Download link not found.')
  if (order.status !== 'paid') throw new AppError('Downloads are not available for this order.', 403, 'DOWNLOAD_UNAVAILABLE')

  // Serve the CURRENT file (admin may have replaced it since purchase); fall back to the purchase-time key.
  const current = await db
    .select({ id: templates.id, r2Key: templates.r2Key })
    .from(templates)
    .where(inArray(templates.id, order.items.map((i) => i.templateId)))
  const keyById = new Map(current.map((t) => [t.id, t.r2Key]))

  const items = await Promise.all(
    order.items.map(async (item) => {
      const key = keyById.get(item.templateId) || item.r2Key
      return { title: item.title, downloadUrl: await presignDownload(key, downloadName(item.title, key), 900) }
    })
  )
  return { orderId: order.id, guestName: order.guestName, items, expiresIn: '15 minutes' }
}

// ── admin ────────────────────────────────────────────────────────────

export async function listAdminOrders(q: z.infer<typeof adminOrdersQuerySchema>) {
  const conditions = []
  if (q.status) conditions.push(eq(orders.status, q.status))
  if (q.search) {
    const like = `%${q.search.replace(/[\\%_]/g, '\\$&')}%`
    conditions.push(
      or(
        ilike(orders.guestName, like),
        ilike(orders.guestEmail, like),
        ilike(orders.razorpayOrderId, like),
        ilike(orders.razorpayPaymentId, like),
        sql`${orders.id}::text ilike ${like}`
      )
    )
  }
  const where = conditions.length ? and(...conditions) : undefined
  const orderBy = {
    newest: desc(orders.createdAt), oldest: asc(orders.createdAt),
    amount_high: desc(orders.totalUsd), amount_low: asc(orders.totalUsd),
  }[q.sort]

  const [{ total }] = await db.select({ total: count() }).from(orders).where(where)
  const data = await db.select().from(orders).where(where).orderBy(orderBy).limit(q.limit).offset(getPaginationOffset(q.page, q.limit))
  return { data, pagination: paginate(q.page, q.limit, Number(total)) }
}

export async function getAdminOrder(id: string): Promise<Order> {
  const [o] = await db.select().from(orders).where(eq(orders.id, id)).limit(1)
  if (!o) throw new NotFoundError('Order not found.')
  return o
}

export async function deleteOrder(id: string) {
  const o = await getAdminOrder(id)
  if (o.status === 'paid' || o.status === 'refunded') {
    throw new AppError('Paid and refunded orders are financial records and cannot be deleted.', 409, 'ORDER_LOCKED')
  }
  await db.delete(orders).where(eq(orders.id, id))
  return { message: 'Order deleted successfully.' }
}

export async function exportOrdersCsv() {
  const rows = await db.select().from(orders).orderBy(desc(orders.createdAt)).limit(20_000)
  const header = ['ID', 'Name', 'Email', 'Items', 'Total USD', 'Currency', 'Amount Charged', 'Exchange Rate', 'Status', 'Method', 'Razorpay Order', 'Razorpay Payment', 'Mode', 'Created', 'Paid']
  const lines = rows.map((o) =>
    [o.id, o.guestName, o.guestEmail, o.items.map((i) => i.title).join('; '), o.totalUsd, o.currencyCharged, o.amountCharged,
    o.exchangeRate ?? '', o.status, o.paymentMethod ?? '', o.razorpayOrderId ?? '', o.razorpayPaymentId ?? '',
    o.livemode ? 'live' : 'test', o.createdAt.toISOString(), o.paidAt?.toISOString() ?? ''].map(csvCell).join(',')
  )
  return [header.map(csvCell).join(','), ...lines].join('\n')
}