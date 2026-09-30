import { FastifyInstance } from 'fastify'
import * as jobsService from './jobs.service.js'
import type {
  CreateJobInput,
  UpdateJobInput,
} from './jobs.service.js'
import { authenticate } from '../../middleware/authenticate.js'
import { requireAdmin } from '@/middleware/require-admin.js'

// ── Public routes ────────────────────────────────────────────────────────────

export async function publicJobRoutes(app: FastifyInstance) {
  // GET /api/jobs — active jobs only
  app.get('/', async (_request, reply) => {
    const data = await jobsService.listActiveJobs()
    return reply.send({ success: true, data })
  })

  // GET /api/jobs/:slug — single job detail
  app.get('/:slug', async (request, reply) => {
    const { slug } = request.params as { slug: string }
    const data = await jobsService.getActiveJobBySlug(slug)
    return reply.send({ success: true, data })
  })
}

// ── Admin routes ─────────────────────────────────────────────────────────────

export async function adminJobRoutes(app: FastifyInstance) {
  app.addHook('onRequest', authenticate)
  app.addHook('onRequest', requireAdmin)

  // GET /api/admin/jobs — all jobs, active + inactive
  app.get('/', async (_request, reply) => {
    const data = await jobsService.listAllJobs()
    return reply.send({ success: true, data })
  })

  // GET /api/admin/jobs/:id
  app.get('/:id', async (request, reply) => {
    const { id } = request.params as { id: string }
    const data = await jobsService.getJobById(id)
    return reply.send({ success: true, data })
  })

  // POST /api/admin/jobs — create
  app.post('/', async (request, reply) => {
    const body = request.body as CreateJobInput
    const data = await jobsService.createJob(body)
    return reply.status(201).send({ success: true, data })
  })

  // PUT /api/admin/jobs/:id — update
  app.put('/:id', async (request, reply) => {
    const { id } = request.params as { id: string }
    const body = request.body as UpdateJobInput
    const data = await jobsService.updateJob(id, body)
    return reply.send({ success: true, data })
  })

  // DELETE /api/admin/jobs/:id
  app.delete('/:id', async (request, reply) => {
    const { id } = request.params as { id: string }
    const data = await jobsService.deleteJob(id)
    return reply.send({ success: true, data })
  })

  // GET /api/admin/jobs/settings/apply-link — get company fallback URL
  app.get('/settings/apply-link', async (_request, reply) => {
    const data = await jobsService.getApplyLinkSettings()
    return reply.send({ success: true, data })
  })

  // PUT /api/admin/jobs/settings/apply-link — set company fallback URL
  app.put('/settings/apply-link', async (request, reply) => {
    const { companyApplyUrl } = request.body as { companyApplyUrl: string | null }
    const data = await jobsService.setApplyLinkSettings(companyApplyUrl)
    return reply.send({ success: true, data })
  })
}