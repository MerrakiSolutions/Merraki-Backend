import {
  pgTable,
  uuid,
  varchar,
  text,
  boolean,
  timestamp,
  pgEnum,
  index,
  inet,
} from "drizzle-orm/pg-core";

export const userRoleEnum = pgEnum("user_role", ["admin", "superadmin"]);

export const activityActionEnum = pgEnum("activity_action", [
  "login",
  "logout",
  "password_change",
  "profile_update",
  "avatar_update",
]);

export const users = pgTable(
  "users",
  {
    id: uuid("id").primaryKey().defaultRandom(),
    name: varchar("name", { length: 255 }).notNull(),
    email: varchar("email", { length: 255 }).notNull().unique(),
    passwordHash: text("password_hash").notNull(),
    avatarUrl: text("avatar_url"),
    role: userRoleEnum("role").default("admin").notNull(),
    isActive: boolean("is_active").default(true).notNull(),
    resetToken: text("reset_token"),
    resetTokenExpires: timestamp("reset_token_expires", { withTimezone: true }),
    inviteToken: text("invite_token"),
    inviteTokenExpires: timestamp("invite_token_expires", {
      withTimezone: true,
    }),
    lastLoginAt: timestamp("last_login_at", { withTimezone: true }),
    createdAt: timestamp("created_at", { withTimezone: true })
      .defaultNow()
      .notNull(),
    updatedAt: timestamp("updated_at", { withTimezone: true })
      .defaultNow()
      .notNull(),
  },
  (t) => ({
    emailIdx: index("users_email_idx").on(t.email),
  }),
);

export const refreshTokens = pgTable(
  "refresh_tokens",
  {
    id: uuid("id").primaryKey().defaultRandom(),
    userId: uuid("user_id")
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
    tokenHash: text("token_hash").notNull(),
    expiresAt: timestamp("expires_at", { withTimezone: true }).notNull(),
    createdAt: timestamp("created_at", { withTimezone: true })
      .defaultNow()
      .notNull(),
  },
  (t) => ({
    userIdIdx: index("refresh_tokens_user_id_idx").on(t.userId),
  }),
);

export const adminActivityLogs = pgTable(
  "admin_activity_logs",
  {
    id: uuid("id").primaryKey().defaultRandom(),
    userId: uuid("user_id")
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
    action: activityActionEnum("action").notNull(),
    ipAddress: inet("ip_address"),
    userAgent: text("user_agent"),
    createdAt: timestamp("created_at", { withTimezone: true })
      .defaultNow()
      .notNull(),
  },
  (t) => ({
    userIdIdx: index("activity_logs_user_id_idx").on(t.userId),
    createdAtIdx: index("activity_logs_created_at_idx").on(t.createdAt),
  }),
);
