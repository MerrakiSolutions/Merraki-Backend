import {
  pgTable,
  uuid,
  varchar,
  text,
  boolean,
  integer,
  timestamp,
  pgEnum,
} from 'drizzle-orm/pg-core'

export const logoTypeEnum = pgEnum('logo_type', [
  'primary',
  'secondary',
  'favicon',
])

export const brandingLogos = pgTable('branding_logos', {
  id: uuid('id').primaryKey().defaultRandom(),
  url: text('url').notNull(),                        // Cloudinary URL
  altText: varchar('alt_text', { length: 255 }),
  type: logoTypeEnum('type').notNull(),
  isActive: boolean('is_active').default(true).notNull(),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
})

export const brandingHeadings = pgTable('branding_headings', {
  id: uuid('id').primaryKey().defaultRandom(),
  page: varchar('page', { length: 100 }).notNull(),  // 'home','store','blog','about'
  heading: varchar('heading', { length: 500 }).notNull(),
  subheading: text('subheading'),
  isActive: boolean('is_active').default(true).notNull(),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
})

export const brandingSocialLinks = pgTable('branding_social_links', {
  id: uuid('id').primaryKey().defaultRandom(),
  platform: varchar('platform', { length: 100 }).notNull(), // 'instagram','linkedin' etc
  url: text('url').notNull(),
  displayOrder: integer('display_order').default(0).notNull(),
  isActive: boolean('is_active').default(true).notNull(),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
})