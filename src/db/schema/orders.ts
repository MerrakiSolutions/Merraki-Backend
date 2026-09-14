import {
  pgTable,
  uuid,
  varchar,
  jsonb,
  numeric,
  timestamp,
  pgEnum,
  inet,
  index,
  text,
} from "drizzle-orm/pg-core";

export const orderStatusEnum = pgEnum("order_status", [
  "pending",
  "paid",
  "failed",
  "refunded",
]);

export const currencyEnum = pgEnum("currency_charged", ["USD", "INR"]);

export const orders = pgTable(
  "orders",
  {
    id: uuid("id").primaryKey().defaultRandom(),

    // guest info
    guestName: varchar("guest_name", { length: 255 }).notNull(),
    guestEmail: varchar("guest_email", { length: 255 }).notNull(),

    // billing
    billingAddress: jsonb("billing_address").notNull(),
    // shape: { line1, line2?, city, state, country, zip, company? }

    // items snapshot — stores title + price at time of purchase
    items: jsonb("items").notNull(),
    // shape: [{ templateId, title, priceUsd, r2Key }]

    // pricing
    subtotalUsd: numeric("subtotal_usd", { precision: 10, scale: 2 }).notNull(),
    totalUsd: numeric("total_usd", { precision: 10, scale: 2 }).notNull(),
    currencyCharged: currencyEnum("currency_charged").notNull(),
    exchangeRate: numeric("exchange_rate", { precision: 10, scale: 4 }),
    amountCharged: numeric("amount_charged", {
      precision: 12,
      scale: 2,
    }).notNull(),

    // razorpay
    razorpayOrderId: varchar("razorpay_order_id", { length: 255 }).unique(),
    razorpayPaymentId: varchar("razorpay_payment_id", { length: 255 }),

    // status
    status: orderStatusEnum("status").default("pending").notNull(),

    // download
    downloadToken: uuid("download_token").defaultRandom().notNull(),

    // invoice
    invoiceR2Key: text("invoice_r2_key"),

    // meta
    ipAddress: inet("ip_address"),
    createdAt: timestamp("created_at", { withTimezone: true })
      .defaultNow()
      .notNull(),
    updatedAt: timestamp("updated_at", { withTimezone: true })
      .defaultNow()
      .notNull(),
  },
  (t) => ({
    guestEmailIdx: index("orders_guest_email_idx").on(t.guestEmail),
    razorpayOrderIdx: index("orders_razorpay_order_idx").on(t.razorpayOrderId),
    downloadTokenIdx: index("orders_download_token_idx").on(t.downloadToken),
    statusIdx: index("orders_status_idx").on(t.status),
  }),
);
