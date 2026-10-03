import { and, eq, gt, isNull, lt } from 'drizzle-orm'
import { db } from '../../db/index.js'
import { orders, type Order } from '../../db/schema/orders.js'
import { env } from '../../config/env.js'
import { escapeHtml as e, sendEmail } from '../payments/payments.lib.js'

const BRAND = '#253957'
const money = (v: string, cur: 'USD' | 'INR') =>
    new Intl.NumberFormat(cur === 'INR' ? 'en-IN' : 'en-US', { style: 'currency', currency: cur }).format(Number(v))

const wrap = (inner: string) => `<div style="background:#f5f7fb;padding:32px 16px;font-family:Arial,sans-serif;color:${BRAND}">
<div style="max-width:560px;margin:auto;background:#fff;border-radius:12px;padding:32px">${inner}
<p style="margin:28px 0 0;font-size:12px;color:#7a96b2">We will never ask for your password or payment details by email.</p></div></div>`

const trackUrl = (id: string) => `${env.FRONTEND_URL.replace(/\/$/, '')}/order-tracking?order_id=${id}`

const button = (href: string, label: string) =>
    `<a href="${e(href)}" style="display:inline-block;background:${BRAND};color:#fff;text-decoration:none;padding:12px 22px;border-radius:8px;font-weight:600">${e(label)}</a>`

function receiptHtml(o: Order) {
    const rows = o.items
        .map((i) => `<tr><td style="padding:8px 0;border-bottom:1px solid #eef1f6">${e(i.title)}</td><td align="right" style="padding:8px 0;border-bottom:1px solid #eef1f6">${money(i.priceUsd, 'USD')}</td></tr>`)
        .join('')
    const charged = o.currencyCharged === 'INR' ? ` (charged ${money(o.amountCharged, 'INR')})` : ''
    return wrap(`<h2 style="margin:0 0 8px">Thanks, ${e(o.guestName.split(' ')[0])} — your files are ready</h2>
<p style="line-height:1.6">Payment received for order #${o.id.slice(0, 8).toUpperCase()}.</p>
<table width="100%" cellpadding="0" cellspacing="0">${rows}
<tr><td style="padding-top:12px;font-weight:700">Total</td><td align="right" style="padding-top:12px;font-weight:700">${money(o.totalUsd, 'USD')}${e(charged)}</td></tr></table>
<p style="margin:24px 0">${button(trackUrl(o.id), 'Download your files')}</p>`)
}

/** Sends the receipt once. Safe to call repeatedly: Resend de-dupes on the idempotency key. */
export async function fulfillOrder(order: Order, opts: { force?: boolean } = {}) {
    if (order.status !== 'paid') return
    if (order.emailSentAt && !opts.force) return
    await sendEmail({
        to: order.guestEmail,
        subject: `Your Merraki order #${order.id.slice(0, 8).toUpperCase()} is ready`,
        html: receiptHtml(order),
        idempotencyKey: opts.force ? undefined : `receipt:${order.id}`,
    })
    await db.update(orders).set({ emailSentAt: new Date() }).where(eq(orders.id, order.id))
}

/** The "queue": every 5 minutes, retry receipts that failed to send (last 3 days only). */
export async function retryUnsentReceipts() {
    const rows = await db
        .select()
        .from(orders)
        .where(
            and(
                eq(orders.status, 'paid'),
                isNull(orders.emailSentAt),
                lt(orders.paidAt, new Date(Date.now() - 60_000)),
                gt(orders.paidAt, new Date(Date.now() - 3 * 86_400_000))
            )
        )
        .limit(20)
    for (const order of rows) {
        try {
            await fulfillOrder(order)
        } catch (err) {
            console.error('receipt retry failed', order.id, err)
        }
    }
}

/** Secure replacement for "look up by email": the links go to the address owner only. */
export async function sendOrderLinks(email: string, list: Pick<Order, 'id' | 'items'>[]) {
    const rows = list
        .map((o) => `<p style="margin:0 0 14px">#${o.id.slice(0, 8).toUpperCase()} — ${e(o.items.map((i) => i.title).join(', ').slice(0, 120))}<br>
<a href="${e(trackUrl(o.id))}" style="color:${BRAND}">Open this order</a></p>`)
        .join('')
    await sendEmail({
        to: email,
        subject: 'Your Merraki order links',
        html: wrap(`<h2 style="margin:0 0 8px">Your orders</h2><p style="line-height:1.6">You (or someone using your email) asked for your order links. If that wasn't you, ignore this email.</p>${rows}`),
    })
}