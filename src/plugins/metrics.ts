import fp from 'fastify-plugin'
import client from 'prom-client'

const { Registry, collectDefaultMetrics, Counter, Histogram } = client

const register = new Registry()

collectDefaultMetrics({
  register,
  prefix: 'merraki_',
})

const httpRequestsTotal = new Counter({
  name: 'merraki_http_requests_total',
  help: 'Total number of HTTP requests',
  labelNames: ['method', 'route', 'status_code'] as const,
  registers: [register],
})

const httpRequestDuration = new Histogram({
  name: 'merraki_http_request_duration_seconds',
  help: 'HTTP request duration in seconds',
  labelNames: ['method', 'route', 'status_code'] as const,
  buckets: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5],
  registers: [register],
})

export default fp(async (app) => {
  app.addHook('onResponse', async (request, reply) => {
    const route = request.routeOptions.url ?? request.url

    httpRequestsTotal.inc({
      method: request.method,
      route,
      status_code: reply.statusCode,
    })

    const duration = reply.elapsedTime / 1000

    httpRequestDuration.observe(
      {
        method: request.method,
        route,
        status_code: reply.statusCode,
      },
      duration,
    )
  })

  app.get('/metrics', async (_request, reply) => {
    reply.header('Content-Type', register.contentType)
    return register.metrics()
  })
})