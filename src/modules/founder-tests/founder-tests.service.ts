import { eq, ilike, or, and, desc, asc, count } from 'drizzle-orm'
import { db } from '../../db/index.js'
import { founderTestResults } from '../../db/schema/founder-tests.js'
import {
    FOUNDER_TEST_QUESTIONS,
    FOUNDER_TEST_RESULTS,
    scoreAnswers,
    validateAnswers,
} from '../../config/founder-test.js'
import { AppError, NotFoundError } from '../../lib/errors.js'
import { paginate, getPaginationOffset } from '../../lib/pagination.js'

// ── PUBLIC: Get test questions ────────────────────────────────────
// Returns questions without scores (scores are backend only)

export const getTestQuestions = () =>
    FOUNDER_TEST_QUESTIONS.map((q) => ({
        id: q.id,
        question: q.question,
        options: q.options.map((o) => ({
            value: o.value,
            label: o.label,
            // score and traits intentionally excluded from public response
        })),
    }))

// ── PUBLIC: Get result types ──────────────────────────────────────
// Returns all possible result archetypes (for landing page preview)

export const getResultTypes = () =>
    FOUNDER_TEST_RESULTS.map((r) => ({
        type: r.type,
        title: r.title,
        description: r.description,
    }))

// ── PUBLIC: Submit test ───────────────────────────────────────────

export const submitTest = async (data: {
    leadName: string
    leadEmail: string
    leadCompany?: string
    answers: Record<string, string>
    ipAddress?: string
}) => {
    const { leadName, leadEmail, leadCompany, answers, ipAddress } = data

    // validate all questions answered with valid values
    const { valid, missing } = validateAnswers(answers)
    if (!valid) {
        throw new AppError(
            `Missing or invalid answers for: ${missing.join(', ')}`,
            400
        )
    }

    // score and compute result
    const { score, resultType } = scoreAnswers(answers)

    // find full result details
    const result = FOUNDER_TEST_RESULTS.find((r) => r.type === resultType)

    // save to DB
    const [created] = await db
        .insert(founderTestResults)
        .values({
            leadName,
            leadEmail,
            leadCompany,
            answers,
            resultType,
            score,
            ipAddress: ipAddress as any,
        })
        .returning({ id: founderTestResults.id })

    return {
        resultId: created.id,
        resultType,
        title: result?.title ?? resultType,
        description: result?.description ?? '',
        score,
    }
}

// ── ADMIN: List all results ───────────────────────────────────────

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
        sort === 'oldest'
            ? asc(founderTestResults.createdAt)
            : desc(founderTestResults.createdAt)

    const [{ total }] = await db
        .select({ total: count() })
        .from(founderTestResults)
        .where(where)

    const data = await db
        .select()
        .from(founderTestResults)
        .where(where)
        .orderBy(orderBy)
        .limit(limit)
        .offset(offset)

    return { data, pagination: paginate(page, limit, Number(total)) }
}

// ── ADMIN: Single result ──────────────────────────────────────────

export const getAdminResultById = async (id: string) => {
    const [result] = await db
        .select()
        .from(founderTestResults)
        .where(eq(founderTestResults.id, id))
        .limit(1)

    if (!result) throw new NotFoundError('Test result not found.')
    return result
}

// ── ADMIN: Delete result ──────────────────────────────────────────

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

// ── ADMIN: Export results CSV ─────────────────────────────────────

export const exportResultsCSV = async () => {
    const data = await db
        .select()
        .from(founderTestResults)
        .orderBy(desc(founderTestResults.createdAt))

    const headers = [
        'ID',
        'Name',
        'Email',
        'Company',
        'Result Type',
        'Score',
        'Created At',
    ]

    const rows = data.map((r) => [
        r.id,
        r.leadName,
        r.leadEmail,
        r.leadCompany || '',
        r.resultType || '',
        String(r.score ?? ''),
        r.createdAt.toISOString(),
    ])

    return [headers, ...rows]
        .map((row) => row.map((v) => `"${v}"`).join(','))
        .join('\n')
}

// ── ADMIN: Stats summary ──────────────────────────────────────────

export const getResultStats = async () => {
    const data = await db
        .select({
            resultType: founderTestResults.resultType,
            total: count(),
        })
        .from(founderTestResults)
        .groupBy(founderTestResults.resultType)

    const totalSubmissions = data.reduce((sum, r) => sum + Number(r.total), 0)

    return {
        totalSubmissions,
        breakdown: data.map((r) => ({
            resultType: r.resultType,
            count: Number(r.total),
            percentage:
                totalSubmissions > 0
                    ? Math.round((Number(r.total) / totalSubmissions) * 100)
                    : 0,
        })),
    }
}