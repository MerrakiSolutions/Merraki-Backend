import { FastifyInstance } from 'fastify'
import * as jobsService from './jobs.service.js'
import { authenticate } from '../../middleware/authenticate.js'
import { requireAdmin } from '../../middleware/require-admin.js'
import {
  idParamSchema,
  slugParamSchema,
  createJobSchema,
  updateJobSchema,
  applyLinkSettingsSchema,
} from './jobs.schema.js'

// ════════════════════════════════════════════════
// PUBLIC — prefix: /api/jobs
// ════════════════════════════════════════════════

export const publicJobRoutes = async (app: FastifyInstance) => {
  app.get('/', async (_request, reply) => {
    const data = await jobsService.listActiveJobs()
    return reply.send({ success: true, data })
  })

  app.get('/:slug', async (request, reply) => {
    const { slug } = slugParamSchema.parse(request.params)
    const data = await jobsService.getActiveJobBySlug(slug)
    return reply.send({ success: true, data })
  })
}

// ════════════════════════════════════════════════
// ADMIN — prefix: /api/admin/jobs
// ════════════════════════════════════════════════

export const adminJobRoutes = async (app: FastifyInstance) => {
  app.addHook('preHandler', authenticate)
  app.addHook('preHandler', requireAdmin)

  app.get('/', async (_request, reply) => {
    const data = await jobsService.listAllJobs()
    return reply.send({ success: true, data })
  })

  app.get('/settings/apply-link', async (_request, reply) => {
    const data = await jobsService.getApplyLinkSettings()
    return reply.send({ success: true, data })
  })

  app.put('/settings/apply-link', async (request, reply) => {
    const { companyApplyUrl } = applyLinkSettingsSchema.parse(request.body)
    const data = await jobsService.setApplyLinkSettings(companyApplyUrl)
    return reply.send({ success: true, data })
  })

  app.get('/:id', async (request, reply) => {
    const { id } = idParamSchema.parse(request.params)
    const data = await jobsService.getJobById(id)
    return reply.send({ success: true, data })
  })

  app.post('/', async (request, reply) => {
    const body = createJobSchema.parse(request.body)
    const data = await jobsService.createJob(body)
    return reply.status(201).send({ success: true, data })
  })

  app.put('/:id', async (request, reply) => {
    const { id } = idParamSchema.parse(request.params)
    const body = updateJobSchema.parse(request.body)
    const data = await jobsService.updateJob(id, body)
    return reply.send({ success: true, data })
  })

  app.delete('/:id', async (request, reply) => {
    const { id } = idParamSchema.parse(request.params)
    const result = await jobsService.deleteJob(id)
    return reply.send({ success: true, ...result })
  })
}