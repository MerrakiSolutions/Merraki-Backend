import { db } from '../../db/index.js'
import { jobs, type Job, type NewJob } from '../../db/schema/jobs.js'
import { brandingSettings } from '../../db/schema/branding.js'
import { eq, asc, desc } from 'drizzle-orm'
import { AppError } from '../../lib/errors.js'
import slugify from "slugify"

// ── Slug helper ────────────────────────────────────────────────────────────────

async function generateUniqueSlug(title: string, excludeId?: string): Promise<string> {
  const baseSlug: string = slugify(title)

  let slug = baseSlug
  let counter = 1

  while (true) {
    const [existing] = await db.select().from(jobs).where(eq(jobs.slug, slug)).limit(1)
    if (!existing || existing.id === excludeId) break
    slug = `${baseSlug}-${counter}`
    counter += 1
  }

  return slug
}

// ── Numeric coercion helper ─────────────────────────────────────────────────────
// displayOrder comes in through JSON/HTTP as `unknown` at runtime even though our
// interfaces declare it as `number` — this guards against a stray string ("3")
// ever reaching Drizzle, which is what produces the
// "Argument of type 'string' is not assignable to parameter of type 'number'" error.

function toDisplayOrder(value: unknown, fallback: number): number {
  if (value === undefined || value === null || value === '') return fallback
  const n = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(n) ? n : fallback
}

// ── Company fallback URL ────────────────────────────────────────────────────────

async function getCompanyApplyUrl(): Promise<string | null> {
  const [settings] = await db.select().from(brandingSettings).limit(1)
  return settings?.companyApplyUrl ?? null
}

function withApplyUrl(job: Job, companyApplyUrl: string | null) {
  return {
    ...job,
    applyUrl: job.applyUrl || companyApplyUrl || null,
  }
}

// ── Public reads ─────────────────────────────────────────────────────────────

export async function listActiveJobs() {
  const activeJobs = await db
    .select()
    .from(jobs)
    .where(eq(jobs.isActive, true))
    .orderBy(asc(jobs.displayOrder), desc(jobs.createdAt))

  const companyApplyUrl = await getCompanyApplyUrl()
  return activeJobs.map((job) => withApplyUrl(job, companyApplyUrl))
}

export async function getActiveJobBySlug(slug: string) {
  const [job] = await db.select().from(jobs).where(eq(jobs.slug, slug)).limit(1)

  if (!job || !job.isActive) {
    throw new AppError('NOT_FOUND', 404)
  }

  const companyApplyUrl = await getCompanyApplyUrl()
  return withApplyUrl(job, companyApplyUrl)
}

// ── Admin reads ──────────────────────────────────────────────────────────────

export async function listAllJobs() {
  return db
    .select()
    .from(jobs)
    .orderBy(asc(jobs.displayOrder), desc(jobs.createdAt))
}

export async function getJobById(id: string): Promise<Job> {
  const [job] = await db.select().from(jobs).where(eq(jobs.id, id)).limit(1)
  if (!job) throw new AppError('NOT_FOUND', 404)
  return job
}

// ── Admin writes ─────────────────────────────────────────────────────────────

export interface CreateJobInput {
  title: string
  team: string
  type: string
  location: string
  description?: unknown
  requirements?: string[]
  applyUrl?: string
  isActive?: boolean
  displayOrder?: number
}

export async function createJob(input: CreateJobInput): Promise<Job> {
  if (!input.title || !input.team || !input.type || !input.location) {
    throw new AppError('VALIDATION_ERROR', 400)
  }

  const slug = await generateUniqueSlug(input.title)

  const values: NewJob = {
    title: input.title,
    slug,
    team: input.team,
    type: input.type,
    location: input.location,
    description: input.description ?? null,
    requirements: input.requirements ?? [],
    applyUrl: input.applyUrl ?? null,
    isActive: input.isActive ?? true,
    displayOrder: toDisplayOrder(input.displayOrder, 0),
  }

  const [created] = await db.insert(jobs).values(values).returning()
  return created
}

export interface UpdateJobInput {
  title?: string
  team?: string
  type?: string
  location?: string
  description?: unknown
  requirements?: string[]
  applyUrl?: string | null
  isActive?: boolean
  displayOrder?: number
}

export async function updateJob(id: string, input: UpdateJobInput): Promise<Job> {
  const existing = await getJobById(id)

  let slug = existing.slug
  if (input.title && input.title !== existing.title) {
    slug = await generateUniqueSlug(input.title, id)
  }

  // Build the patch explicitly rather than spreading `input` directly —
  // this guarantees displayOrder is a real number before it ever reaches
  // Drizzle's `.set()`, which is where the type error was surfacing.
  const patch: Partial<NewJob> = {
    ...(input.title !== undefined && { title: input.title }),
    ...(input.team !== undefined && { team: input.team }),
    ...(input.type !== undefined && { type: input.type }),
    ...(input.location !== undefined && { location: input.location }),
    ...(input.description !== undefined && { description: input.description }),
    ...(input.requirements !== undefined && { requirements: input.requirements }),
    ...(input.applyUrl !== undefined && { applyUrl: input.applyUrl }),
    ...(input.isActive !== undefined && { isActive: input.isActive }),
    ...(input.displayOrder !== undefined && {
      displayOrder: toDisplayOrder(input.displayOrder, existing.displayOrder),
    }),
    slug,
    updatedAt: new Date(),
  }

  const [updated] = await db
    .update(jobs)
    .set(patch)
    .where(eq(jobs.id, id))
    .returning()

  return updated
}

export async function deleteJob(id: string): Promise<{ id: string }> {
  const [deleted] = await db.delete(jobs).where(eq(jobs.id, id)).returning()
  if (!deleted) throw new AppError('NOT_FOUND', 404)
  return { id: deleted.id }
}

// ── Company-wide apply link settings ─────────────────────────────────────────

export async function getApplyLinkSettings() {
  const companyApplyUrl = await getCompanyApplyUrl()
  return { companyApplyUrl }
}

export async function setApplyLinkSettings(companyApplyUrl: string | null) {
  const [existing] = await db.select().from(brandingSettings).limit(1)

  if (existing) {
    const [updated] = await db
      .update(brandingSettings)
      .set({ companyApplyUrl, updatedAt: new Date() })
      .where(eq(brandingSettings.id, existing.id))
      .returning()
    return updated
  }

  const [created] = await db
    .insert(brandingSettings)
    .values({ companyApplyUrl })
    .returning()
  return created
}