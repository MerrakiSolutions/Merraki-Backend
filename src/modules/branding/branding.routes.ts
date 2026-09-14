import { FastifyInstance } from 'fastify'
import * as brandingService from './branding.service.js'
import { authenticate } from '../../middleware/authenticate.js'
import { requireAdmin } from '../../middleware/require-admin.js'
import {
    createLogoSchema,
    updateLogoSchema,
    createHeadingSchema,
    updateHeadingSchema,
    createSocialLinkSchema,
    updateSocialLinkSchema,
    reorderSocialLinksSchema,
} from './branding.schema.js'
import { AppError } from '../../lib/errors.js'

// ════════════════════════════════════════════════
// PUBLIC ROUTES
// prefix: /api/branding
// ════════════════════════════════════════════════

export const publicBrandingRoutes = async (app: FastifyInstance) => {

    // GET /api/branding
    // All active branding in one call — optional ?page= filter for headings
    app.get('/', async (request, reply) => {
        const { page } = request.query as { page?: string }
        const data = await brandingService.getPublicBranding(page)
        return reply.send({ success: true, data })
    })

    // GET /api/branding/logos
    app.get('/logos', async (_request, reply) => {
        const data = await brandingService.getAllLogos(true)
        return reply.send({ success: true, data })
    })

    // GET /api/branding/headings
    app.get('/headings', async (request, reply) => {
        const { page } = request.query as { page?: string }
        const data = await brandingService.getAllHeadings(page, true)
        return reply.send({ success: true, data })
    })

    // GET /api/branding/social-links
    app.get('/social-links', async (_request, reply) => {
        const data = await brandingService.getAllSocialLinks(true)
        return reply.send({ success: true, data })
    })
}

// ════════════════════════════════════════════════
// ADMIN ROUTES
// prefix: /api/admin/branding
// ════════════════════════════════════════════════

export const adminBrandingRoutes = async (app: FastifyInstance) => {

    app.addHook('preHandler', authenticate)
    app.addHook('preHandler', requireAdmin)

    // ── LOGOS ─────────────────────────────────────
    // Logo create/update uses multipart — image upload required

    app.get('/logos', async (_request, reply) => {
        const data = await brandingService.getAllLogos(false)
        return reply.send({ success: true, data })
    })

    app.get('/logos/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const data = await brandingService.getLogoById(id)
        return reply.send({ success: true, data })
    })

    app.post('/logos', async (request, reply) => {
        const parts = request.parts()
        const fields: Record<string, string> = {}
        let imageFile: { buffer: Buffer; mimetype: string } | undefined

        for await (const part of parts) {
            if (part.type === 'file' && part.fieldname === 'image') {
                imageFile = { buffer: await part.toBuffer(), mimetype: part.mimetype }
            } else if (part.type !== 'file') {
                fields[part.fieldname] = part.value as string
            }
        }

        if (!imageFile) throw new AppError('Logo image file is required.', 400)

        const body = createLogoSchema.parse({
            altText: fields.altText,
            type: fields.type,
            isActive: fields.isActive !== undefined ? fields.isActive === 'true' : true,
        })

        const data = await brandingService.createLogo({ ...body, imageFile })
        return reply.status(201).send({ success: true, data })
    })

    app.put('/logos/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const parts = request.parts()
        const fields: Record<string, string> = {}
        let imageFile: { buffer: Buffer; mimetype: string } | undefined

        for await (const part of parts) {
            if (part.type === 'file' && part.fieldname === 'image') {
                imageFile = { buffer: await part.toBuffer(), mimetype: part.mimetype }
            } else if (part.type !== 'file') {
                fields[part.fieldname] = part.value as string
            }
        }

        const body = updateLogoSchema.parse({
            altText: fields.altText,
            type: fields.type,
            isActive: fields.isActive !== undefined
                ? fields.isActive === 'true'
                : undefined,
        })

        const data = await brandingService.updateLogo(id, { ...body, imageFile })
        return reply.send({ success: true, data })
    })

    app.delete('/logos/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const result = await brandingService.deleteLogo(id)
        return reply.send({ success: true, ...result })
    })

    // ── HEADINGS ──────────────────────────────────
    // Headings are plain JSON — no file upload needed

    app.get('/headings', async (request, reply) => {
        const { page } = request.query as { page?: string }
        const data = await brandingService.getAllHeadings(page, false)
        return reply.send({ success: true, data })
    })

    app.get('/headings/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const data = await brandingService.getHeadingById(id)
        return reply.send({ success: true, data })
    })

    app.post('/headings', async (request, reply) => {
        const body = createHeadingSchema.parse(request.body)
        const data = await brandingService.createHeading(body)
        return reply.status(201).send({ success: true, data })
    })

    app.put('/headings/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const body = updateHeadingSchema.parse(request.body)
        const data = await brandingService.updateHeading(id, body)
        return reply.send({ success: true, data })
    })

    app.delete('/headings/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const result = await brandingService.deleteHeading(id)
        return reply.send({ success: true, ...result })
    })

    // ── SOCIAL LINKS ──────────────────────────────
    // Social links are plain JSON — no file upload needed

    app.get('/social-links', async (_request, reply) => {
        const data = await brandingService.getAllSocialLinks(false)
        return reply.send({ success: true, data })
    })

    app.get('/social-links/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const data = await brandingService.getSocialLinkById(id)
        return reply.send({ success: true, data })
    })

    app.post('/social-links', async (request, reply) => {
        const body = createSocialLinkSchema.parse(request.body)
        const data = await brandingService.createSocialLink(body)
        return reply.status(201).send({ success: true, data })
    })

    app.put('/social-links/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const body = updateSocialLinkSchema.parse(request.body)
        const data = await brandingService.updateSocialLink(id, body)
        return reply.send({ success: true, data })
    })

    app.delete('/social-links/:id', async (request, reply) => {
        const { id } = request.params as { id: string }
        const result = await brandingService.deleteSocialLink(id)
        return reply.send({ success: true, ...result })
    })

    // PUT /api/admin/branding/social-links/reorder
    // Bulk reorder — admin drags to sort, frontend sends new order
    app.put('/social-links/reorder', async (request, reply) => {
        const body = reorderSocialLinksSchema.parse(request.body)
        const result = await brandingService.reorderSocialLinks(body.items)
        return reply.send({ success: true, ...result })
    })
}