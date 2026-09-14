import postgres from 'postgres'
import { drizzle } from 'drizzle-orm/postgres-js'
import { users } from './schema/users.js'
import { eq } from 'drizzle-orm'
import bcrypt from 'bcryptjs'
import dotenv from 'dotenv'

dotenv.config()

const client = postgres(process.env.DATABASE_URL!, { max: 1 })
const db = drizzle(client)

async function seed() {
  const email = process.env.SUPERADMIN_EMAIL!
  const password = process.env.SUPERADMIN_PASSWORD!
  const name = process.env.SUPERADMIN_NAME!

  const existing = await db
    .select()
    .from(users)
    .where(eq(users.email, email))

  if (existing.length > 0) {
    console.log('✅ Superadmin already exists, skipping.')
    await client.end()
    return
  }

  const passwordHash = await bcrypt.hash(password, 12)

  await db.insert(users).values({
    name,
    email,
    passwordHash,
    role: 'superadmin',
    isActive: true,
  })

  console.log(`✅ Superadmin created: ${email}`)
  await client.end()
}

seed().catch((err) => {
  console.error('❌ Seed failed:', err)
  process.exit(1)
})