import Fastify from 'fastify'
import cors from '@fastify/cors'
import cookie from '@fastify/cookie'
import multipart from '@fastify/multipart'
import rateLimit from '@fastify/rate-limit'

import { env } from './config/env.js'
import { AppError } from './lib/errors.js'

// ── Route imports ─────────────────────────────────────────────────
import { authRoutes } from './modules/auth/auth.routes.js'
import { publicTemplateRoutes, adminTemplateRoutes } from './modules/templates/templates.routes.js'
import { checkoutRoutes } from './modules/checkout/checkout.routes.js'
import { paymentsRoutes } from './modules/payments/payments.routes.js'
import { publicOrderRoutes, adminOrderRoutes } from './modules/orders/orders.routes.js'
import { publicBlogRoutes, adminBlogRoutes } from './modules/blog/blog.routes.js'
import { publicNewsletterRoutes, adminNewsletterRoutes } from './modules/newsletter/newsletter.routes.js'
import { publicContactRoutes, adminContactRoutes } from './modules/contacts/contacts.routes.js'
import { publicFounderTestRoutes, adminFounderTestRoutes } from './modules/founder-tests/founder-tests.routes.js'
import { publicBrandingRoutes, adminBrandingRoutes } from './modules/branding/branding.routes.js'
import { dashboardRoutes } from './modules/dashboard/dashboard.routes.js'

const app = Fastify({
  logger: {
    level: env.NODE_ENV === 'development' ? 'info' : 'warn',
    transport:
      env.NODE_ENV === 'development'
        ? { target: 'pino-pretty' }
        : undefined,
  },
  trustProxy: true,
})

// ── Plugins ───────────────────────────────────────────────────────

await app.register(cors, {
  origin: [env.FRONTEND_URL],
  credentials: true,
  methods: ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS'],
})

await app.register(cookie, {
  secret: env.JWT_REFRESH_SECRET,
})

await app.register(multipart, {
  limits: { fileSize: 50 * 1024 * 1024, files: 10 },
})

await app.register(rateLimit, {
  max: 100,
  timeWindow: '1 minute',
})

// ── Raw body for Razorpay webhook ─────────────────────────────────

app.addContentTypeParser(
  'application/json',
  { parseAs: 'string' },
  (_req, body, done) => {
    try {
      ; (_req as any).rawBody = body as string
      done(null, JSON.parse(body as string))
    } catch (err) {
      done(err as Error, undefined)
    }
  }
)

// ── Global error handler ──────────────────────────────────────────

app.setErrorHandler((error, _request, reply) => {
  if (error instanceof AppError) {
    return reply.status(error.statusCode).send({
      success: false,
      error: { code: error.code, message: error.message },
    })
  }

  if (error instanceof Error && 'validation' in error && error.validation) {
    return reply.status(400).send({
      success: false,
      error: {
        code: 'VALIDATION_ERROR',
        message: 'Validation failed',
        details: (error as any).validation,
      },
    })
  }

  app.log.error(error)
  return reply.status(500).send({
    success: false,
    error: { code: 'INTERNAL_ERROR', message: 'Something went wrong' },
  })
})

// ── Health check ──────────────────────────────────────────────────

app.get('/health', async () => ({
  status: 'ok',
  timestamp: new Date().toISOString(),
  environment: env.NODE_ENV,
}))

// ════════════════════════════════════════════════
// ROUTES
// ════════════════════════════════════════════════

// admin auth + account management
await app.register(authRoutes, { prefix: '/api/admin/auth' })

// templates
await app.register(publicTemplateRoutes, { prefix: '/api' })
await app.register(adminTemplateRoutes, { prefix: '/api/admin/templates' })

// checkout + payments + orders
await app.register(checkoutRoutes, { prefix: '/api/checkout' })
await app.register(paymentsRoutes, { prefix: '/api/payments' })
await app.register(publicOrderRoutes, { prefix: '/api/orders' })
await app.register(adminOrderRoutes, { prefix: '/api/admin/orders' })

// blog
await app.register(publicBlogRoutes, { prefix: '/api/blog' })
await app.register(adminBlogRoutes, { prefix: '/api/admin/blog' })

// newsletter
await app.register(publicNewsletterRoutes, { prefix: '/api/newsletter' })
await app.register(adminNewsletterRoutes, { prefix: '/api/admin/newsletter' })

// contacts
await app.register(publicContactRoutes, { prefix: '/api/contacts' })
await app.register(adminContactRoutes, { prefix: '/api/admin/contacts' })

// founder test
await app.register(publicFounderTestRoutes, { prefix: '/api/founder-test' })
await app.register(adminFounderTestRoutes, { prefix: '/api/admin/founder-test' })

// branding
await app.register(publicBrandingRoutes, { prefix: '/api/branding' })
await app.register(adminBrandingRoutes, { prefix: '/api/admin/branding' })

// dashboard
await app.register(dashboardRoutes, { prefix: '/api/admin/dashboard' })

// ── Start ─────────────────────────────────────────────────────────

const start = async () => {
  try {
    await app.listen({ port: Number(env.PORT), host: '0.0.0.0' })
    console.log(`🚀 Server running on port ${env.PORT}`)
  } catch (err) {
    app.log.error(err)
    process.exit(1)
  }
}

start()

export default app