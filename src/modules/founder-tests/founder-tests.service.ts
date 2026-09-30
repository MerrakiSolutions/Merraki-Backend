import { eq, ilike, or, and, desc, asc, count } from 'drizzle-orm'
import { db } from '../../db/index.js'
import { founderTestResults } from '../../db/schema/founder-tests.js'
import {
    founderTestQuestions,
    type NewFounderTestQuestion,
} from '../../db/schema/founder-test-questions.js'
import {
    founderTestArchetypes,
    type NewFounderTestArchetype,
} from '../../db/schema/founder-test-archetypes.js'
import { AppError, NotFoundError } from '../../lib/errors.js'
import { paginate, getPaginationOffset } from '../../lib/pagination.js'

// ── PUBLIC: questions (no scores exposed) ──────────────────────────

export const getPublicQuestions = async () => {
    const questions = await db
        .select()
        .from(founderTestQuestions)
        .where(eq(founderTestQuestions.isActive, true))
        .orderBy(asc(founderTestQuestions.displayOrder))

    return questions.map((q) => ({
        id: q.questionKey,
        section: q.section,
        sectionLabel: q.sectionLabel,
        category: q.category,
        question: q.question,
        description: q.description,
        type: q.type,
        options: (q.options as { value: string; label: string }[]).map((o) => ({
            value: o.value,
            label: o.label,
            // score intentionally excluded
        })),
    }))
}

// ── PUBLIC: submit ──────────────────────────────────────────────────

export const submitTest = async (data: {
    leadName: string
    leadEmail: string
    leadCompany?: string
    answers: Record<string, string>
    ipAddress?: string
}) => {
    const { leadName, leadEmail, leadCompany, answers, ipAddress } = data

    const activeQuestions = await db
        .select()
        .from(founderTestQuestions)
        .where(eq(founderTestQuestions.isActive, true))
        .orderBy(asc(founderTestQuestions.displayOrder))

    if (activeQuestions.length === 0) {
        throw new AppError('No active questions configured for this test.', 500)
    }

    // ── validate ──
    const missing: string[] = []
    for (const q of activeQuestions) {
        const answer = answers[q.questionKey]
        const validValues = (q.options as { value: string }[]).map((o) => o.value)
        if (!answer || !validValues.includes(answer)) {
            missing.push(q.questionKey)
        }
    }
    if (missing.length > 0) {
        throw new AppError(`Missing or invalid answers for: ${missing.join(', ')}`, 400)
    }

    // ── score ──
    let totalScore = 0
    const sectionMap = new Map<string, { label: string; score: number; max: number }>()

    for (const q of activeQuestions) {
        const selected = answers[q.questionKey]
        const option = (q.options as { value: string; score: number }[]).find(
            (o) => o.value === selected
        )
        const points = option?.score ?? 0
        totalScore += points

        const bucket = sectionMap.get(q.section) ?? { label: q.sectionLabel, score: 0, max: 0 }
        bucket.score += points
        bucket.max += 10
        sectionMap.set(q.section, bucket)
    }

    const sectionScores = Array.from(sectionMap.entries()).map(([key, v]) => ({
        dimension: key,
        label: v.label,
        score: v.score,
        max: v.max,
        percentage: v.max > 0 ? Math.round((v.score / v.max) * 100) : 0,
    }))

    // ── match archetype ──
    const archetypes = await db
        .select()
        .from(founderTestArchetypes)
        .where(eq(founderTestArchetypes.isActive, true))
        .orderBy(asc(founderTestArchetypes.displayOrder))

    const matched =
        archetypes.find((a) => totalScore >= a.minScore && totalScore <= a.maxScore) ??
        archetypes[Math.floor(archetypes.length / 2)] ?? null

    const resultType = matched?.archetypeKey ?? 'unclassified'

    // ── save lead ──
    const [created] = await db
        .insert(founderTestResults)
        .values({
            leadName,
            leadEmail,
            leadCompany,
            answers,
            resultType,
            score: totalScore,
            ipAddress: ipAddress as any,
        })
        .returning({ id: founderTestResults.id })

    return {
        resultId: created.id,
        resultType,
        title: matched?.title ?? 'Result',
        badge: matched?.badge ?? '',
        color: matched?.color ?? '#000000',
        description: matched?.description ?? '',
        message: matched?.message ?? '',
        traits: (matched?.traits as string[]) ?? [],
        strengths: (matched?.strengths as string[]) ?? [],
        growthSuggestions: (matched?.growthSuggestions as string[]) ?? [],
        riskAreas: (matched?.riskAreas as string[]) ?? [],
        score: totalScore,
        totalMax: activeQuestions.length * 10,
        sectionScores,
    }
}

// ── ADMIN: questions CRUD ────────────────────────────────────────────

export const getAdminQuestions = async () => {
    return db
        .select()
        .from(founderTestQuestions)
        .orderBy(asc(founderTestQuestions.displayOrder))
}

export const getAdminQuestionById = async (id: string) => {
    const [q] = await db
        .select()
        .from(founderTestQuestions)
        .where(eq(founderTestQuestions.id, id))
        .limit(1)
    if (!q) throw new NotFoundError('Question not found.')
    return q
}

export const createQuestion = async (input: {
    questionKey: string
    section: string
    sectionLabel: string
    category: string
    question: string
    description?: string
    type: 'single' | 'scale'
    options: { value: string; label: string; score: number }[]
    displayOrder?: number
    isActive?: boolean
}) => {
    const [existing] = await db
        .select({ id: founderTestQuestions.id })
        .from(founderTestQuestions)
        .where(eq(founderTestQuestions.questionKey, input.questionKey))
        .limit(1)

    if (existing) {
        throw new AppError(`Question key "${input.questionKey}" already exists.`, 409)
    }

    if (!input.options || input.options.length < 2) {
        throw new AppError('A question needs at least 2 options.', 400)
    }

    const values: NewFounderTestQuestion = {
        questionKey: input.questionKey,
        section: input.section,
        sectionLabel: input.sectionLabel,
        category: input.category,
        question: input.question,
        description: input.description ?? null,
        type: input.type,
        options: input.options,
        displayOrder: input.displayOrder ?? 0,
        isActive: input.isActive ?? true,
    }

    const [created] = await db.insert(founderTestQuestions).values(values).returning()
    return created
}

export const updateQuestion = async (
    id: string,
    input: Partial<{
        section: string
        sectionLabel: string
        category: string
        question: string
        description: string | null
        type: 'single' | 'scale'
        options: { value: string; label: string; score: number }[]
        displayOrder: number
        isActive: boolean
    }>
) => {
    await getAdminQuestionById(id)

    if (input.options && input.options.length < 2) {
        throw new AppError('A question needs at least 2 options.', 400)
    }

    const [updated] = await db
        .update(founderTestQuestions)
        .set({ ...input, updatedAt: new Date() })
        .where(eq(founderTestQuestions.id, id))
        .returning()

    return updated
}

export const deleteQuestion = async (id: string) => {
    await getAdminQuestionById(id)
    await db.delete(founderTestQuestions).where(eq(founderTestQuestions.id, id))
    return { message: 'Question deleted successfully.' }
}

export const reorderQuestions = async (items: { id: string; displayOrder: number }[]) => {
    for (const item of items) {
        await db
            .update(founderTestQuestions)
            .set({ displayOrder: item.displayOrder, updatedAt: new Date() })
            .where(eq(founderTestQuestions.id, item.id))
    }
    return { message: 'Order updated.' }
}

// ── ADMIN: archetypes CRUD ───────────────────────────────────────────

export const getAdminArchetypes = async () => {
    return db
        .select()
        .from(founderTestArchetypes)
        .orderBy(asc(founderTestArchetypes.displayOrder))
}

export const getAdminArchetypeById = async (id: string) => {
    const [a] = await db
        .select()
        .from(founderTestArchetypes)
        .where(eq(founderTestArchetypes.id, id))
        .limit(1)
    if (!a) throw new NotFoundError('Archetype not found.')
    return a
}

export const createArchetype = async (input: {
    archetypeKey: string
    title: string
    badge?: string
    color?: string
    description: string
    message: string
    traits?: string[]
    strengths?: string[]
    growthSuggestions?: string[]
    riskAreas?: string[]
    minScore: number
    maxScore: number
    displayOrder?: number
    isActive?: boolean
}) => {
    const [existing] = await db
        .select({ id: founderTestArchetypes.id })
        .from(founderTestArchetypes)
        .where(eq(founderTestArchetypes.archetypeKey, input.archetypeKey))
        .limit(1)

    if (existing) {
        throw new AppError(`Archetype key "${input.archetypeKey}" already exists.`, 409)
    }

    if (input.minScore > input.maxScore) {
        throw new AppError('minScore cannot be greater than maxScore.', 400)
    }

    const values: NewFounderTestArchetype = {
        archetypeKey: input.archetypeKey,
        title: input.title,
        badge: input.badge ?? null,
        color: input.color ?? null,
        description: input.description,
        message: input.message,
        traits: input.traits ?? [],
        strengths: input.strengths ?? [],
        growthSuggestions: input.growthSuggestions ?? [],
        riskAreas: input.riskAreas ?? [],
        minScore: input.minScore,
        maxScore: input.maxScore,
        displayOrder: input.displayOrder ?? 0,
        isActive: input.isActive ?? true,
    }

    const [created] = await db.insert(founderTestArchetypes).values(values).returning()
    return created
}

export const updateArchetype = async (
    id: string,
    input: Partial<{
        title: string
        badge: string | null
        color: string | null
        description: string
        message: string
        traits: string[]
        strengths: string[]
        growthSuggestions: string[]
        riskAreas: string[]
        minScore: number
        maxScore: number
        displayOrder: number
        isActive: boolean
    }>
) => {
    const existing = await getAdminArchetypeById(id)

    const minScore = input.minScore ?? existing.minScore
    const maxScore = input.maxScore ?? existing.maxScore
    if (minScore > maxScore) {
        throw new AppError('minScore cannot be greater than maxScore.', 400)
    }

    const [updated] = await db
        .update(founderTestArchetypes)
        .set({ ...input, updatedAt: new Date() })
        .where(eq(founderTestArchetypes.id, id))
        .returning()

    return updated
}

export const deleteArchetype = async (id: string) => {
    await getAdminArchetypeById(id)
    await db.delete(founderTestArchetypes).where(eq(founderTestArchetypes.id, id))
    return { message: 'Archetype deleted successfully.' }
}

// ── ADMIN: leads (existing — unchanged logic, kept here for completeness) ──

export const getAdminResults = async (query: {
    page: number
    limit: number
    search?: string
    resultType?: string
    sort: string
}) => {
    const { page, limit, search, resultType, sort } = query
    const offset = getPaginationOffset(page, limit)

    const conditions = []
    if (search) {
        conditions.push(
            or(
                ilike(founderTestResults.leadName, `%${search}%`),
                ilike(founderTestResults.leadEmail, `%${search}%`),
                ilike(founderTestResults.leadCompany, `%${search}%`)
            )
        )
    }
    if (resultType) {
        conditions.push(eq(founderTestResults.resultType, resultType))
    }

    const where = conditions.length > 0 ? and(...conditions) : undefined
    const orderBy =
        sort === 'oldest' ? asc(founderTestResults.createdAt) : desc(founderTestResults.createdAt)

    const [{ total }] = await db.select({ total: count() }).from(founderTestResults).where(where)

    const data = await db
        .select()
        .from(founderTestResults)
        .where(where)
        .orderBy(orderBy)
        .limit(limit)
        .offset(offset)

    return { data, pagination: paginate(page, limit, Number(total)) }
}

export const getAdminResultById = async (id: string) => {
    const [result] = await db
        .select()
        .from(founderTestResults)
        .where(eq(founderTestResults.id, id))
        .limit(1)
    if (!result) throw new NotFoundError('Test result not found.')
    return result
}

export const deleteResult = async (id: string) => {
    const [existing] = await db
        .select({ id: founderTestResults.id })
        .from(founderTestResults)
        .where(eq(founderTestResults.id, id))
        .limit(1)
    if (!existing) throw new NotFoundError('Test result not found.')

    await db.delete(founderTestResults).where(eq(founderTestResults.id, id))
    return { message: 'Result deleted successfully.' }
}

export const exportResultsCSV = async () => {
    const data = await db
        .select()
        .from(founderTestResults)
        .orderBy(desc(founderTestResults.createdAt))

    const headers = ['ID', 'Name', 'Email', 'Company', 'Result Type', 'Score', 'Created At']
    const rows = data.map((r) => [
        r.id,
        r.leadName,
        r.leadEmail,
        r.leadCompany || '',
        r.resultType || '',
        String(r.score ?? ''),
        r.createdAt.toISOString(),
    ])

    return [headers, ...rows].map((row) => row.map((v) => `"${v}"`).join(',')).join('\n')
}

export const getResultStats = async () => {
    const data = await db
        .select({ resultType: founderTestResults.resultType, total: count() })
        .from(founderTestResults)
        .groupBy(founderTestResults.resultType)

    const totalSubmissions = data.reduce((sum, r) => sum + Number(r.total), 0)

    return {
        totalSubmissions,
        breakdown: data.map((r) => ({
            resultType: r.resultType,
            count: Number(r.total),
            percentage: totalSubmissions > 0 ? Math.round((Number(r.total) / totalSubmissions) * 100) : 0,
        })),
    }
}