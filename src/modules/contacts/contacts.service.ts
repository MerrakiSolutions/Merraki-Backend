import { eq, ilike, or, and, desc, asc, count } from 'drizzle-orm'
import { db } from '../../db/index.js'
import { contacts } from '../../db/schema/contacts.js'
import { NotFoundError } from '../../lib/errors.js'
import { paginate, getPaginationOffset } from '../../lib/pagination.js'

// ── PUBLIC: Submit contact form ───────────────────────────────────

export const submitContact = async (data: {
    name: string
    email: string
    phone?: string
    message: string
    ipAddress?: string
}) => {
    const [created] = await db
        .insert(contacts)
        .values({
            name: data.name,
            email: data.email,
            phone: data.phone,
            message: data.message,
        })
        .returning({ id: contacts.id })

    return {
        message: "Thanks for reaching out! We'll get back to you within 24 hours.",
        id: created.id,
    }
}

// ── ADMIN: List contacts ──────────────────────────────────────────

export const getAdminContacts = async (query: {
    page: number
    limit: number
    search?: string
    status?: string
    sort: string
}) => {
    const { page, limit, search, status, sort } = query
    const offset = getPaginationOffset(page, limit)

    const conditions = []

    if (status) {
        conditions.push(eq(contacts.status, status as any))
    }

    if (search) {
        conditions.push(
            or(
                ilike(contacts.name, `%${search}%`),
                ilike(contacts.email, `%${search}%`),
                ilike(contacts.message, `%${search}%`)
            )
        )
    }

    const where = conditions.length > 0 ? and(...conditions) : undefined
    const orderBy = sort === 'oldest' ? asc(contacts.createdAt) : desc(contacts.createdAt)

    const [{ total }] = await db
        .select({ total: count() })
        .from(contacts)
        .where(where)

    const data = await db
        .select()
        .from(contacts)
        .where(where)
        .orderBy(orderBy)
        .limit(limit)
        .offset(offset)

    return { data, pagination: paginate(page, limit, Number(total)) }
}

// ── ADMIN: Single contact ─────────────────────────────────────────

export const getAdminContactById = async (id: string) => {
    const [contact] = await db
        .select()
        .from(contacts)
        .where(eq(contacts.id, id))
        .limit(1)

    if (!contact) throw new NotFoundError('Contact not found.')
    return contact
}

// ── ADMIN: Update contact status ──────────────────────────────────

export const updateContactStatus = async (
    id: string,
    status: 'new' | 'read' | 'replied' | 'archived'
) => {
    const [existing] = await db
        .select({ id: contacts.id })
        .from(contacts)
        .where(eq(contacts.id, id))
        .limit(1)

    if (!existing) throw new NotFoundError('Contact not found.')

    const [updated] = await db
        .update(contacts)
        .set({ status, updatedAt: new Date() })
        .where(eq(contacts.id, id))
        .returning()

    return updated
}

// ── ADMIN: Delete contact ─────────────────────────────────────────

export const deleteContact = async (id: string) => {
    const [existing] = await db
        .select({ id: contacts.id })
        .from(contacts)
        .where(eq(contacts.id, id))
        .limit(1)

    if (!existing) throw new NotFoundError('Contact not found.')

    await db.delete(contacts).where(eq(contacts.id, id))
    return { message: 'Contact deleted successfully.' }
}

// ── ADMIN: Export contacts CSV ────────────────────────────────────

export const exportContactsCSV = async () => {
    const data = await db
        .select()
        .from(contacts)
        .orderBy(desc(contacts.createdAt))

    const headers = ['ID', 'Name', 'Email', 'Phone', 'Message', 'Status', 'Created At']

    const rows = data.map((c) => [
        c.id,
        c.name,
        c.email,
        c.phone || '',
        c.message.replace(/"/g, '""'),   // escape quotes in message
        c.status,
        c.createdAt.toISOString(),
    ])

    return [headers, ...rows]
        .map((row) => row.map((v) => `"${v}"`).join(','))
        .join('\n')
}