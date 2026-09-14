import {
  pgTable,
  uuid,
  varchar,
  jsonb,
  integer,
  timestamp,
  inet,
  index,
} from 'drizzle-orm/pg-core'

export const founderTestResults = pgTable(
  'founder_test_results',
  {
    id: uuid('id').primaryKey().defaultRandom(),
    leadName: varchar('lead_name', { length: 255 }).notNull(),
    leadEmail: varchar('lead_email', { length: 255 }).notNull(),
    leadCompany: varchar('lead_company', { length: 255 }),
    answers: jsonb('answers').notNull(),       // { q1: "A", q2: "B", ... }
    resultType: varchar('result_type', { length: 255 }),
    score: integer('score'),
    ipAddress: inet('ip_address'),
    createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  },
  (t) => ({
    emailIdx: index('founder_test_results_email_idx').on(t.leadEmail),
    resultTypeIdx: index('founder_test_results_result_type_idx').on(t.resultType),
  })
)