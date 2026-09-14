import {
  pgTable,
  uuid,
  varchar,
  text,
  timestamp,
  pgEnum,
  index,
} from 'drizzle-orm/pg-core'

export const contactStatusEnum = pgEnum('contact_status', [
  'new',
  'read',
  'replied',
  'archived',
])

export const contacts = pgTable(
  'contacts',
  {
    id: uuid('id').primaryKey().defaultRandom(),
    name: varchar('name', { length: 255 }).notNull(),
    email: varchar('email', { length: 255 }).notNull(),
    phone: varchar('phone', { length: 50 }),
    message: text('message').notNull(),
    status: contactStatusEnum('status').default('new').notNull(),
    createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
    updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
  },
  (t) => ({
    statusIdx: index('contacts_status_idx').on(t.status),
    emailIdx: index('contacts_email_idx').on(t.email),
  })
)