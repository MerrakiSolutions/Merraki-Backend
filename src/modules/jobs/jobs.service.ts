import { db } from '../../db/index.js'
import { jobs, type Job, type NewJob } from '../../db/schema/jobs.js'
import { brandingSettings } from '../../db/schema/branding.js'
import { eq, asc, desc } from 'drizzle-orm'
import { NotFoundError } from '../../lib/errors.js'
import slugify from 'slugify'
import type { CreateJobInput, UpdateJobInput } from './jobs.schema.js'

// ── Slug helper ──────────────────────────────────────────────────────────────

async function generateUniqueSlug(title: string): Promise<string> {
  const baseSlug = slugify(title, { lower: true, strict: true }) || 'job'
  let slug = baseSlug
  let counter = 1

  while (true) {
    const [existing] = await db
      .select({ id: jobs.id })
      .from(jobs)
      .where(eq(jobs.slug, slug))
      .limit(1)
    if (!existing) break
    counter += 1
    slug = `${baseSlug}-${counter}`
  }

  return slug
}

// ── Company fallback link ────────────────────────────────────────────────────

async function getCompanyApplyUrl(): Promise<string | null> {
  const [settings] = await db.select().from(brandingSettings).limit(1)
  return settings?.companyApplyUrl ?? null
}

// Public responses carry the RESOLVED link: job link → company link → null
function withResolvedApplyUrl(job: Job, companyApplyUrl: string | null) {
  return { ...job, applyUrl: job.applyUrl || companyApplyUrl || null }
}

// ── Public reads ─────────────────────────────────────────────────────────────

export async function listActiveJobs() {
  const activeJobs = await db
    .select()
    .from(jobs)
    .where(eq(jobs.isActive, true))
    .orderBy(asc(jobs.displayOrder), desc(jobs.createdAt))

  const companyApplyUrl = await getCompanyApplyUrl()
  return activeJobs.map((job) => withResolvedApplyUrl(job, companyApplyUrl))
}

export async function getActiveJobBySlug(slug: string) {
  const [job] = await db.select().from(jobs).where(eq(jobs.slug, slug)).limit(1)
  if (!job || !job.isActive) throw new NotFoundError('Job not found.')

  const companyApplyUrl = await getCompanyApplyUrl()
  return withResolvedApplyUrl(job, companyApplyUrl)
}

// ── Admin reads (raw rows — admin needs to see the job's OWN link) ───────────

export async function listAllJobs() {
  return db.select().from(jobs).orderBy(asc(jobs.displayOrder), desc(jobs.createdAt))
}

export async function getJobById(id: string): Promise<Job> {
  const [job] = await db.select().from(jobs).where(eq(jobs.id, id)).limit(1)
  if (!job) throw new NotFoundError('Job not found.')
  return job
}

// ── Admin writes ─────────────────────────────────────────────────────────────

export async function createJob(input: CreateJobInput): Promise<Job> {
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
    displayOrder: input.displayOrder ?? 0,
  }

  const [created] = await db.insert(jobs).values(values).returning()
  return created
}

export async function updateJob(id: string, input: UpdateJobInput): Promise<Job> {
  await getJobById(id)

  // Slug stays stable on edit so shared/bookmarked job links never break
  const [updated] = await db
    .update(jobs)
    .set({ ...input, updatedAt: new Date() })
    .where(eq(jobs.id, id))
    .returning()

  return updated
}

export async function deleteJob(id: string): Promise<{ message: string }> {
  const [deleted] = await db.delete(jobs).where(eq(jobs.id, id)).returning({ id: jobs.id })
  if (!deleted) throw new NotFoundError('Job not found.')
  return { message: 'Job deleted successfully.' }
}

// ── Company-wide apply link ──────────────────────────────────────────────────

export async function getApplyLinkSettings() {
  return { companyApplyUrl: await getCompanyApplyUrl() }
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

  const [created] = await db.insert(brandingSettings).values({ companyApplyUrl }).returning()
  return created
}