import { pgTable, uuid, varchar, jsonb, boolean, integer, timestamp } from 'drizzle-orm/pg-core'

export const jobs = pgTable('jobs', {
  id: uuid('id').primaryKey().defaultRandom(),
  title: varchar('title', { length: 200 }).notNull(),
  slug: varchar('slug', { length: 220 }).notNull().unique(),
  team: varchar('team', { length: 100 }).notNull(),
  type: varchar('type', { length: 50 }).notNull(),
  location: varchar('location', { length: 150 }).notNull(),
  description: jsonb('description'),
  requirements: jsonb('requirements').$type<string[]>().default([]),
  applyUrl: varchar('apply_url', { length: 500 }),   // renamed: generic external link
  isActive: boolean('is_active').notNull().default(true),
  displayOrder: integer('display_order').notNull().default(0),
  createdAt: timestamp('created_at').notNull().defaultNow(),
  updatedAt: timestamp('updated_at').notNull().defaultNow(),
})

export type Job = typeof jobs.$inferSelect
export type NewJob = typeof jobs.$inferInsert