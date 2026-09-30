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

export interface QuestionOption {
    value: string   // 'a', 'b', 'c', 'd' or '1'..'10' for scale
    label: string
    score: number   // 0-10, never sent to public
}

export const founderTestQuestions = pgTable('founder_test_questions', {
    id: uuid('id').primaryKey().defaultRandom(),
    questionKey: varchar('question_key', { length: 50 }).notNull().unique(), // "q1"
    section: varchar('section', { length: 100 }).notNull(),
    sectionLabel: varchar('section_label', { length: 150 }).notNull(),
    category: varchar('category', { length: 200 }).notNull(),
    question: text('question').notNull(),
    description: text('description'),
    type: varchar('type', { length: 20 }).notNull().default('single'), // 'single' | 'scale'
    options: jsonb('options').$type<QuestionOption[]>().notNull().default([]),
    displayOrder: integer('display_order').notNull().default(0),
    isActive: boolean('is_active').notNull().default(true),
    createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
    updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
})

export type FounderTestQuestion = typeof founderTestQuestions.$inferSelect
export type NewFounderTestQuestion = typeof founderTestQuestions.$inferInsert