import {
  pgTable, uuid, varchar, jsonb, numeric, timestamp, pgEnum, inet, index, text, boolean,
} from "drizzle-orm/pg-core";

export const orderStatusEnum = pgEnum("order_status", ["pending", "paid", "failed", "refunded"]);
export const currencyEnum = pgEnum("currency_charged", ["USD", "INR"]);

export interface OrderItem {
  templateId: string;
  title: string;
  priceUsd: string; // "29.99"
  r2Key: string; // internal — never sent to the public
}

export interface BillingAddress {
  line1: string;
  line2?: string;
  city: string;
  state: string;
  country: string;
  zip: string;
  company?: string;
}

export const orders = pgTable(
  "orders",
  {
    id: uuid("id").primaryKey().defaultRandom(),

    guestName: varchar("guest_name", { length: 255 }).notNull(),
    guestEmail: varchar("guest_email", { length: 255 }).notNull(),

    billingAddress: jsonb("billing_address").$type<BillingAddress>().notNull(),
    items: jsonb("items").$type<OrderItem[]>().notNull(),

    subtotalUsd: numeric("subtotal_usd", { precision: 10, scale: 2 }).notNull(),
    totalUsd: numeric("total_usd", { precision: 10, scale: 2 }).notNull(),
    currencyCharged: currencyEnum("currency_charged").notNull(),
    exchangeRate: numeric("exchange_rate", { precision: 10, scale: 4 }),
    amountCharged: numeric("amount_charged", { precision: 12, scale: 2 }).notNull(),

    razorpayOrderId: varchar("razorpay_order_id", { length: 255 }).unique(),
    razorpayPaymentId: varchar("razorpay_payment_id", { length: 255 }),

    status: orderStatusEnum("status").default("pending").notNull(),

    downloadToken: uuid("download_token").defaultRandom().notNull(),
    invoiceR2Key: text("invoice_r2_key"),
    ipAddress: inet("ip_address"),

    // ── new ──
    paymentMethod: varchar("payment_method", { length: 30 }),
    paidAt: timestamp("paid_at", { withTimezone: true }),
    emailSentAt: timestamp("email_sent_at", { withTimezone: true }),
    livemode: boolean("livemode").notNull().default(true),

    createdAt: timestamp("created_at", { withTimezone: true }).defaultNow().notNull(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).defaultNow().notNull(),
  },
  (t) => ({
    guestEmailIdx: index("orders_guest_email_idx").on(t.guestEmail),
    razorpayOrderIdx: index("orders_razorpay_order_idx").on(t.razorpayOrderId),
    downloadTokenIdx: index("orders_download_token_idx").on(t.downloadToken),
    statusIdx: index("orders_status_idx").on(t.status),
  }),
);

export type Order = typeof orders.$inferSelect;
export type NewOrder = typeof orders.$inferInsert;