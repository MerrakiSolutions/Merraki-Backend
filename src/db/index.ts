import { drizzle } from "drizzle-orm/postgres-js";
import postgres from "postgres";
import { env } from "../config/env.js";
import * as usersSchema from "./schema/users.js";
import * as blogSchema from "./schema/blog.js";
import * as templatesSchema from "./schema/templates.js";
import * as ordersSchema from "./schema/orders.js";
import * as founderTestsSchema from "./schema/founder-tests.js";
import * as contactsSchema from "./schema/contacts.js";
import * as newsletterSchema from "./schema/newsletter.js";
import * as brandingSchema from "./schema/branding.js";

const client = postgres(env.DATABASE_URL, {
  max: 10,
  idle_timeout: 20,
  connect_timeout: 10,
});

export const db = drizzle(client, {
  schema: {
    ...usersSchema,
    ...blogSchema,
    ...templatesSchema,
    ...ordersSchema,
    ...founderTestsSchema,
    ...contactsSchema,
    ...newsletterSchema,
    ...brandingSchema,
  },
});

export type DB = typeof db;
