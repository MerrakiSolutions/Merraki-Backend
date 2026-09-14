import {
  pgTable,
  uuid,
  varchar,
  boolean,
  text,
  jsonb,
  timestamp,
  pgEnum,
  index,
} from "drizzle-orm/pg-core";

export const campaignStatusEnum = pgEnum("campaign_status", [
  "draft",
  "sending",
  "sent",
]);

export const newsletterCategories = pgTable("newsletter_categories", {
  id: uuid("id").primaryKey().defaultRandom(),
  name: varchar("name", { length: 255 }).notNull(),
  slug: varchar("slug", { length: 255 }).notNull().unique(),
  createdAt: timestamp("created_at", { withTimezone: true })
    .defaultNow()
    .notNull(),
  updatedAt: timestamp("updated_at", { withTimezone: true })
    .defaultNow()
    .notNull(),
});

export const newsletterSubscribers = pgTable(
  "newsletter_subscribers",
  {
    id: uuid("id").primaryKey().defaultRandom(),
    email: varchar("email", { length: 255 }).notNull().unique(),
    name: varchar("name", { length: 255 }),
    categoryIds: uuid("category_ids").array().default([]).notNull(),
    confirmed: boolean("confirmed").default(false).notNull(),
    confirmToken: text("confirm_token"),
    unsubscribeToken: text("unsubscribe_token").notNull(),
    createdAt: timestamp("created_at", { withTimezone: true })
      .defaultNow()
      .notNull(),
    updatedAt: timestamp("updated_at", { withTimezone: true })
      .defaultNow()
      .notNull(),
  },
  (t) => ({
    emailIdx: index("newsletter_subscribers_email_idx").on(t.email),
    confirmTokenIdx: index("newsletter_subscribers_confirm_token_idx").on(
      t.confirmToken,
    ),
    unsubscribeTokenIdx: index(
      "newsletter_subscribers_unsubscribe_token_idx",
    ).on(t.unsubscribeToken),
  }),
);

export const newsletterCampaigns = pgTable(
  "newsletter_campaigns",
  {
    id: uuid("id").primaryKey().defaultRandom(),
    title: varchar("title", { length: 500 }).notNull(),
    subject: varchar("subject", { length: 500 }).notNull(),
    content: jsonb("content").notNull(), // TipTap JSON
    categoryId: uuid("category_id").references(() => newsletterCategories.id, {
      onDelete: "set null",
    }), // null = send to ALL confirmed subscribers
    status: campaignStatusEnum("status").default("draft").notNull(),
    sentAt: timestamp("sent_at", { withTimezone: true }),
    recipientCount: varchar("recipient_count", { length: 20 }).default("0"),
    createdAt: timestamp("created_at", { withTimezone: true })
      .defaultNow()
      .notNull(),
    updatedAt: timestamp("updated_at", { withTimezone: true })
      .defaultNow()
      .notNull(),
  },
  (t) => ({
    statusIdx: index("newsletter_campaigns_status_idx").on(t.status),
  }),
);
