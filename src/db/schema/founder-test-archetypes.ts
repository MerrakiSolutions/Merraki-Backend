import {
    pgTable,
    uuid,
    varchar,
    text,
    jsonb,
    integer,
    boolean,
    timestamp,
} from 'drizzle-orm/pg-core'

export const founderTestArchetypes = pgTable('founder_test_archetypes', {
    id: uuid('id').primaryKey().defaultRandom(),
    archetypeKey: varchar('archetype_key', { length: 50 }).notNull().unique(),
    title: varchar('title', { length: 150 }).notNull(),
    badge: varchar('badge', { length: 10 }),
    color: varchar('color', { length: 20 }),
    description: text('description').notNull(),
    message: text('message').notNull(),
    traits: jsonb('traits').$type<string[]>().notNull().default([]),
    strengths: jsonb('strengths').$type<string[]>().notNull().default([]),
    growthSuggestions: jsonb('growth_suggestions').$type<string[]>().notNull().default([]),
    riskAreas: jsonb('risk_areas').$type<string[]>().notNull().default([]),
    minScore: integer('min_score').notNull(),
    maxScore: integer('max_score').notNull(),
    displayOrder: integer('display_order').notNull().default(0),
    isActive: boolean('is_active').notNull().default(true),
    createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
    updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
})

export type FounderTestArchetype = typeof founderTestArchetypes.$inferSelect
export type NewFounderTestArchetype = typeof founderTestArchetypes.$inferInsert