import {
  pgTable,
  uuid,
  varchar,
  text,
  jsonb,
  numeric,
  boolean,
  timestamp,
  pgEnum,
  index,
} from "drizzle-orm/pg-core";

export const templateStatusEnum = pgEnum("template_status", [
  "draft",
  "published",
  "archived",
]);

export const templateCategories = pgTable("template_categories", {
  id: uuid("id").primaryKey().defaultRandom(),
  name: varchar("name", { length: 255 }).notNull(),
  slug: varchar("slug", { length: 255 }).notNull().unique(),
  description: text("description"),
  createdAt: timestamp("created_at", { withTimezone: true })
    .defaultNow()
    .notNull(),
  updatedAt: timestamp("updated_at", { withTimezone: true })
    .defaultNow()
    .notNull(),
});

export const templates = pgTable(
  "templates",
  {
    id: uuid("id").primaryKey().defaultRandom(),
    title: varchar("title", { length: 500 }).notNull(),
    slug: varchar("slug", { length: 500 }).notNull().unique(),
    description: text("description"),
    longDescription: jsonb("long_description"), // TipTap JSON
    priceUsd: numeric("price_usd", { precision: 10, scale: 2 }).notNull(),
    r2Key: text("r2_key").notNull(), // private R2 object key
    previewImages: jsonb("preview_images").default("[]"), // [{ url, alt }]
    categoryId: uuid("category_id").references(() => templateCategories.id, {
      onDelete: "set null",
    }),
    tags: text("tags").array().default([]),
    featured: boolean("featured").default(false).notNull(),
    status: templateStatusEnum("status").default("draft").notNull(),
    createdAt: timestamp("created_at", { withTimezone: true })
      .defaultNow()
      .notNull(),
    updatedAt: timestamp("updated_at", { withTimezone: true })
      .defaultNow()
      .notNull(),
  },
  (t) => ({
    slugIdx: index("templates_slug_idx").on(t.slug),
    statusIdx: index("templates_status_idx").on(t.status),
    categoryIdx: index("templates_category_idx").on(t.categoryId),
    featuredIdx: index("templates_featured_idx").on(t.featured),
  }),
);
