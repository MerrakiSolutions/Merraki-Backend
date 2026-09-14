import { eq, asc } from 'drizzle-orm'
import { db } from '../../db/index.js'
import {
    brandingLogos,
    brandingHeadings,
    brandingSocialLinks,
} from '../../db/schema/branding.js'
import { uploadImageToCloudinary } from '../../lib/cloudinary.js'
import { AppError, NotFoundError } from '../../lib/errors.js'

// ════════════════════════════════════════════════
// LOGOS
// ════════════════════════════════════════════════

export const getAllLogos = async (activeOnly = false) => {
    const data = await db
        .select()
        .from(brandingLogos)
        .orderBy(asc(brandingLogos.createdAt))

    return activeOnly ? data.filter((l) => l.isActive) : data
}

export const getLogoById = async (id: string) => {
    const [logo] = await db
        .select()
        .from(brandingLogos)
        .where(eq(brandingLogos.id, id))
        .limit(1)

    if (!logo) throw new NotFoundError('Logo not found.')
    return logo
}

export const createLogo = async (data: {
    altText?: string
    type: 'primary' | 'secondary' | 'favicon'
    isActive?: boolean
    imageFile: { buffer: Buffer; mimetype: string }
}) => {
    const allowed = ['image/jpeg', 'image/png', 'image/webp', 'image/svg+xml', 'image/x-icon']
    if (!allowed.includes(data.imageFile.mimetype)) {
        throw new AppError('Only JPEG, PNG, WebP, SVG, and ICO files are allowed for logos.', 400)
    }

    const { url } = await uploadImageToCloudinary(
        data.imageFile.buffer,
        'merraki/branding/logos',
        `logo_${data.type}_${Date.now()}`
    )

    const [created] = await db
        .insert(brandingLogos)
        .values({
            url,
            altText: data.altText,
            type: data.type,
            isActive: data.isActive ?? true,
        })
        .returning()

    return created
}

export const updateLogo = async (
    id: string,
    data: {
        altText?: string
        type?: 'primary' | 'secondary' | 'favicon'
        isActive?: boolean
        imageFile?: { buffer: Buffer; mimetype: string }
    }
) => {
    const [existing] = await db
        .select()
        .from(brandingLogos)
        .where(eq(brandingLogos.id, id))
        .limit(1)

    if (!existing) throw new NotFoundError('Logo not found.')

    const updateData: Record<string, any> = { updatedAt: new Date() }

    if (data.altText !== undefined) updateData.altText = data.altText
    if (data.type !== undefined) updateData.type = data.type
    if (data.isActive !== undefined) updateData.isActive = data.isActive

    if (data.imageFile) {
        const allowed = ['image/jpeg', 'image/png', 'image/webp', 'image/svg+xml', 'image/x-icon']
        if (!allowed.includes(data.imageFile.mimetype)) {
            throw new AppError('Only JPEG, PNG, WebP, SVG, and ICO files are allowed for logos.', 400)
        }
        const { url } = await uploadImageToCloudinary(
            data.imageFile.buffer,
            'merraki/branding/logos',
            `logo_${existing.type}_${id}`
        )
        updateData.url = url
    }

    const [updated] = await db
        .update(brandingLogos)
        .set(updateData)
        .where(eq(brandingLogos.id, id))
        .returning()

    return updated
}

export const deleteLogo = async (id: string) => {
    const [existing] = await db
        .select({ id: brandingLogos.id })
        .from(brandingLogos)
        .where(eq(brandingLogos.id, id))
        .limit(1)

    if (!existing) throw new NotFoundError('Logo not found.')

    await db.delete(brandingLogos).where(eq(brandingLogos.id, id))
    return { message: 'Logo deleted successfully.' }
}

// ════════════════════════════════════════════════
// HEADINGS
// ════════════════════════════════════════════════

export const getAllHeadings = async (page?: string, activeOnly = false) => {
    const data = await db
        .select()
        .from(brandingHeadings)
        .orderBy(asc(brandingHeadings.page))

    let filtered = data
    if (page) filtered = filtered.filter((h) => h.page === page)
    if (activeOnly) filtered = filtered.filter((h) => h.isActive)

    return filtered
}

export const getHeadingById = async (id: string) => {
    const [heading] = await db
        .select()
        .from(brandingHeadings)
        .where(eq(brandingHeadings.id, id))
        .limit(1)

    if (!heading) throw new NotFoundError('Heading not found.')
    return heading
}

export const createHeading = async (data: {
    page: string
    heading: string
    subheading?: string
    isActive?: boolean
}) => {
    const [created] = await db
        .insert(brandingHeadings)
        .values({
            page: data.page,
            heading: data.heading,
            subheading: data.subheading,
            isActive: data.isActive ?? true,
        })
        .returning()

    return created
}

export const updateHeading = async (
    id: string,
    data: {
        page?: string
        heading?: string
        subheading?: string
        isActive?: boolean
    }
) => {
    const [existing] = await db
        .select({ id: brandingHeadings.id })
        .from(brandingHeadings)
        .where(eq(brandingHeadings.id, id))
        .limit(1)

    if (!existing) throw new NotFoundError('Heading not found.')

    const updateData: Record<string, any> = { updatedAt: new Date() }
    if (data.page !== undefined) updateData.page = data.page
    if (data.heading !== undefined) updateData.heading = data.heading
    if (data.subheading !== undefined) updateData.subheading = data.subheading
    if (data.isActive !== undefined) updateData.isActive = data.isActive

    const [updated] = await db
        .update(brandingHeadings)
        .set(updateData)
        .where(eq(brandingHeadings.id, id))
        .returning()

    return updated
}

export const deleteHeading = async (id: string) => {
    const [existing] = await db
        .select({ id: brandingHeadings.id })
        .from(brandingHeadings)
        .where(eq(brandingHeadings.id, id))
        .limit(1)

    if (!existing) throw new NotFoundError('Heading not found.')

    await db.delete(brandingHeadings).where(eq(brandingHeadings.id, id))
    return { message: 'Heading deleted successfully.' }
}

// ════════════════════════════════════════════════
// SOCIAL LINKS
// ════════════════════════════════════════════════

export const getAllSocialLinks = async (activeOnly = false) => {
    const data = await db
        .select()
        .from(brandingSocialLinks)
        .orderBy(asc(brandingSocialLinks.displayOrder))

    return activeOnly ? data.filter((s) => s.isActive) : data
}

export const getSocialLinkById = async (id: string) => {
    const [link] = await db
        .select()
        .from(brandingSocialLinks)
        .where(eq(brandingSocialLinks.id, id))
        .limit(1)

    if (!link) throw new NotFoundError('Social link not found.')
    return link
}

export const createSocialLink = async (data: {
    platform: string
    url: string
    displayOrder?: number
    isActive?: boolean
}) => {
    const [created] = await db
        .insert(brandingSocialLinks)
        .values({
            platform: data.platform,
            url: data.url,
            displayOrder: data.displayOrder ?? 0,
            isActive: data.isActive ?? true,
        })
        .returning()

    return created
}

export const updateSocialLink = async (
    id: string,
    data: {
        platform?: string
        url?: string
        displayOrder?: number
        isActive?: boolean
    }
) => {
    const [existing] = await db
        .select({ id: brandingSocialLinks.id })
        .from(brandingSocialLinks)
        .where(eq(brandingSocialLinks.id, id))
        .limit(1)

    if (!existing) throw new NotFoundError('Social link not found.')

    const updateData: Record<string, any> = { updatedAt: new Date() }
    if (data.platform !== undefined) updateData.platform = data.platform
    if (data.url !== undefined) updateData.url = data.url
    if (data.displayOrder !== undefined) updateData.displayOrder = data.displayOrder
    if (data.isActive !== undefined) updateData.isActive = data.isActive

    const [updated] = await db
        .update(brandingSocialLinks)
        .set(updateData)
        .where(eq(brandingSocialLinks.id, id))
        .returning()

    return updated
}

export const deleteSocialLink = async (id: string) => {
    const [existing] = await db
        .select({ id: brandingSocialLinks.id })
        .from(brandingSocialLinks)
        .where(eq(brandingSocialLinks.id, id))
        .limit(1)

    if (!existing) throw new NotFoundError('Social link not found.')

    await db.delete(brandingSocialLinks).where(eq(brandingSocialLinks.id, id))
    return { message: 'Social link deleted successfully.' }
}

export const reorderSocialLinks = async (
    items: { id: string; displayOrder: number }[]
) => {
    // update each link's display order
    await Promise.all(
        items.map((item) =>
            db
                .update(brandingSocialLinks)
                .set({ displayOrder: item.displayOrder, updatedAt: new Date() })
                .where(eq(brandingSocialLinks.id, item.id))
        )
    )

    return { message: 'Social links reordered successfully.' }
}

// ════════════════════════════════════════════════
// PUBLIC: Get all branding in one call
// Frontend calls this once on load
// ════════════════════════════════════════════════

export const getPublicBranding = async (page?: string) => {
    const [logos, headings, socialLinks] = await Promise.all([
        getAllLogos(true),
        getAllHeadings(page, true),
        getAllSocialLinks(true),
    ])

    return { logos, headings, socialLinks }
}