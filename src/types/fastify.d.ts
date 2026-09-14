import { AccessTokenPayload } from '../lib/jwt.js'

declare module 'fastify' {
  interface FastifyRequest {
    user?: AccessTokenPayload
    rawBody?: string
  }
}