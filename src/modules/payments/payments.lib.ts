import { createHmac, timingSafeEqual } from 'node:crypto'
import { isIP } from 'node:net'
import Razorpay from 'razorpay'
import { S3Client, GetObjectCommand } from '@aws-sdk/client-s3'
import { getSignedUrl } from '@aws-sdk/s3-request-presigner'
import type { FastifyRequest } from 'fastify'
import { env } from '../../config/env.js'
import { AppError } from '../../lib/errors.js'

// ── Razorpay ─────────────────────────────────────────────────────────
export const razorpay = new Razorpay({ key_id: env.RAZORPAY_KEY_ID, key_secret: env.RAZORPAY_KEY_SECRET })
export const isLiveMode = env.RAZORPAY_KEY_ID.startsWith('rzp_live_')
console.log(`Razorpay: ${isLiveMode ? 'LIVE' : 'TEST'} mode`)

// The SDK throws plain objects, not Errors
export const describeRazorpayError = (err: unknown): string =>
    (err as any)?.error?.description ?? (err as any)?.message ?? 'Payment provider error'

// ── Money: exact decimal-string ⇄ minor units (no float math) ────────
export function toMinor(value: string | number): number {
    const s = typeof value === 'number' ? value.toFixed(2) : String(value).trim()
    const m = /^(\d+)(?:\.(\d+))?$/.exec(s)
    if (!m) throw new Error(`Invalid money value: "${s}"`)
    return Number(m[1]) * 100 + Number((m[2] ?? '').padEnd(2, '0').slice(0, 2))
}
export const fromMinor = (minor: number) => `${Math.floor(minor / 100)}.${String(minor % 100).padStart(2, '0')}`

// ── Signatures ───────────────────────────────────────────────────────
const hmac = (secret: string, data: string) => createHmac('sha256', secret).update(data).digest('hex')
export function safeEqual(a: string, b: string) {
    const x = Buffer.from(a), y = Buffer.from(b)
    return x.length === y.length && timingSafeEqual(x, y)
}
export const verifyCheckoutSignature = (orderId: string, paymentId: string, sig: string) =>
    safeEqual(hmac(env.RAZORPAY_KEY_SECRET, `${orderId}|${paymentId}`), sig.toLowerCase())
export const verifyWebhookSignature = (rawBody: string, sig: string) =>
    safeEqual(hmac(env.RAZORPAY_WEBHOOK_SECRET, rawBody), sig.toLowerCase())

// ── Request helpers ──────────────────────────────────────────────────
export const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

/** Cloudflare sits in front of the API — prefer its header. */
export function getClientIp(req: FastifyRequest): string | null {
    const cf = req.headers['cf-connecting-ip']
    const ip = (Array.isArray(cf) ? cf[0] : cf) ?? req.ip
    return ip && isIP(ip) ? ip : null
}
export const rateLimit = (max: number, timeWindow: string) => ({
    config: { rateLimit: { max, timeWindow, keyGenerator: (r: FastifyRequest) => getClientIp(r) ?? r.ip } },
})

// ── Text safety (we email these strings, so no phishing payloads) ────
export const sanitizeText = (s: string, max = 120) =>
    s.replace(/[\u0000-\u001F\u007F\u200B-\u200F\u202A-\u202E\u2060\uFEFF<>]/g, '').replace(/\s+/g, ' ').trim().slice(0, max)
export const looksLikeLink = (s: string) =>
    /(https?:\/\/|www\.|@|\b[\w-]+\.(com|net|org|io|co|in|xyz|top|ru|cn|info|biz|link|click)\b)/i.test(s)
export const escapeHtml = (s: string) =>
    s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;')
export function csvCell(v: unknown) {
    let s = v == null ? '' : String(v)
    if (/^[=+\-@\t\r]/.test(s)) s = `'${s}` // spreadsheet formula injection
    return `"${s.replace(/"/g, '""')}"`
}

// ── USD → INR (cached 1h, stale up to 24h, sanity-checked) ───────────
let fx: { rate: number; at: number } | null = null
export async function getUsdInr(): Promise<number> {
    if (fx && Date.now() - fx.at < 3_600_000) return fx.rate
    try {
        const res = await fetch(
            `https://v6.exchangerate-api.com/v6/${env.EXCHANGE_RATE_API_KEY}/pair/USD/INR`,
            { signal: AbortSignal.timeout(5000) }
        )
        const json = (await res.json()) as { result?: string; conversion_rate?: number }
        const rate = Number(Number(json.conversion_rate).toFixed(4))
        if (!res.ok || json.result !== 'success' || !(rate > 50 && rate < 200)) throw new Error('bad FX response')
        fx = { rate, at: Date.now() }
        return rate
    } catch {
        if (fx && Date.now() - fx.at < 24 * 3_600_000) return fx.rate
        throw new AppError('INR payments are temporarily unavailable. Please try again shortly.', 503, 'FX_UNAVAILABLE')
    }
}

// ── R2 signed downloads ──────────────────────────────────────────────
const s3 = new S3Client({
    region: 'auto',
    endpoint: `https://${env.R2_ACCOUNT_ID}.r2.cloudflarestorage.com`,
    credentials: { accessKeyId: env.R2_ACCESS_KEY_ID, secretAccessKey: env.R2_SECRET_ACCESS_KEY },
})
export const presignDownload = (key: string, filename: string, expiresIn = 900) =>
    getSignedUrl(
        s3,
        new GetObjectCommand({
            Bucket: env.R2_BUCKET_NAME,
            Key: key,
            ResponseContentDisposition: `attachment; filename="${filename.replace(/"/g, '')}"`,
        }),
        { expiresIn }
    )

// ── Email (Resend REST — idempotency key de-dupes retries for 24h) ───
export async function sendEmail(input: { to: string; subject: string; html: string; idempotencyKey?: string }) {
    const res = await fetch('https://api.resend.com/emails', {
        method: 'POST',
        headers: {
            Authorization: `Bearer ${env.RESEND_API_KEY}`,
            'Content-Type': 'application/json',
            ...(input.idempotencyKey ? { 'Idempotency-Key': input.idempotencyKey } : {}),
        },
        body: JSON.stringify({
            from: `${env.RESEND_FROM_NAME.replace(/["<>]/g, '')} <${env.RESEND_FROM_EMAIL}>`,
            to: input.to,
            subject: input.subject,
            html: input.html,
        }),
        signal: AbortSignal.timeout(15_000),
    })
    if (!res.ok) throw new Error(`Resend ${res.status}: ${(await res.text().catch(() => '')).slice(0, 200)}`)
}

/** Best-effort email to you for things that need a human. Alerts go to SUPERADMIN_EMAIL. */
export async function alertAdmin(subject: string, lines: string[]) {
    console.warn(`[ALERT] ${subject}`, lines)
    try {
        await sendEmail({
            to: env.SUPERADMIN_EMAIL,
            subject: `[Merraki ALERT] ${subject}`,
            html: `<h3>${escapeHtml(subject)}</h3><ul>${lines.map((l) => `<li>${escapeHtml(l)}</li>`).join('')}</ul>`,
        })
    } catch (err) {
        console.error('alert email failed', err)
    }
}