/**
 * Dedicated seed data for screenshot generation.
 *
 * Provides realistic gates and requests that showcase mroki's core
 * use cases: API refactoring, database migrations, and framework upgrades.
 */

import type { ApiHelper } from './fixtures'

// ── Gate definitions ────────────────────────────────────────────────

interface GateDef {
  name: string
  liveUrl: string
  shadowUrl: string
}

export const gates: Record<string, GateDef> = {
  orders: {
    name: 'orders-api',
    liveUrl: 'https://api.acme.io/v1/orders',
    shadowUrl: 'https://api-canary.acme.io/v1/orders',
  },
  users: {
    name: 'users-service',
    liveUrl: 'https://api.acme.io/v1/users',
    shadowUrl: 'https://api-next.acme.io/v2/users',
  },
  payments: {
    name: 'payments-gateway',
    liveUrl: 'https://payments.acme.io/charges',
    shadowUrl: 'https://payments-beta.acme.io/charges',
  },
}

// ── Realistic response bodies ───────────────────────────────────────

const orderLive = {
  id: 'ord_4a8fZ9',
  status: 'confirmed',
  customer: { id: 'cust_9k3m', name: 'Alice Johnson', tier: 'premium' },
  items: [
    { sku: 'WIDGET-001', name: 'Standard Widget', qty: 2, unit_price: 29.99 },
    { sku: 'GADGET-042', name: 'Pro Gadget', qty: 1, unit_price: 149.0 },
  ],
  totals: { subtotal: 208.98, tax: 18.81, total: 227.79, currency: 'USD' },
  shipping: { method: 'express', estimated_days: 2, tracking_id: null },
  created_at: '2026-03-28T14:32:00Z',
}

const orderShadow = {
  id: 'ord_4a8fZ9',
  status: 'confirmed',
  customer: { id: 'cust_9k3m', name: 'Alice Johnson', tier: 'premium' },
  items: [
    { sku: 'WIDGET-001', name: 'Standard Widget', qty: 2, unit_price: 29.99 },
    { sku: 'GADGET-042', name: 'Pro Gadget', qty: 1, unit_price: 149.0 },
    { sku: 'PROMO-100', name: 'Welcome Gift', qty: 1, unit_price: 0.0 },
  ],
  totals: {
    subtotal: 208.98,
    discount: 15.0,
    tax: 17.46,
    total: 211.44,
    currency: 'USD',
  },
  shipping: { method: 'express', estimated_days: 2, tracking_id: 'TRK-8829' },
  created_at: '2026-03-28T14:32:00Z',
}

const orderDiffOps = [
  { op: 'add', path: '/body/items/2', value: { sku: 'PROMO-100', name: 'Welcome Gift', qty: 1, unit_price: 0.0 } },
  { op: 'add', path: '/body/totals/discount', value: 15.0 },
  { op: 'replace', path: '/body/totals/tax', value: 17.46 },
  { op: 'replace', path: '/body/totals/total', value: 211.44 },
  { op: 'replace', path: '/body/shipping/tracking_id', value: 'TRK-8829' },
]

// ── Request definitions for the orders gate ─────────────────────────

interface RequestDef {
  method: string
  path: string
  liveBody: object
  shadowBody: object
  liveStatus: number
  shadowStatus: number
  diffOps: { op: string; path: string; value?: unknown }[]
}

const ordersRequests: RequestDef[] = [
  {
    method: 'POST',
    path: '/v1/orders',
    liveBody: orderLive,
    shadowBody: orderShadow,
    liveStatus: 201,
    shadowStatus: 201,
    diffOps: orderDiffOps,
  },
  {
    method: 'GET',
    path: '/v1/orders/ord_4a8fZ9',
    liveBody: orderLive,
    shadowBody: orderLive,
    liveStatus: 200,
    shadowStatus: 200,
    diffOps: [],
  },
  {
    method: 'PUT',
    path: '/v1/orders/ord_4a8fZ9/confirm',
    liveBody: orderLive,
    shadowBody: orderShadow,
    liveStatus: 200,
    shadowStatus: 200,
    diffOps: orderDiffOps,
  },
  {
    method: 'GET',
    path: '/v1/orders?page=1&limit=20',
    liveBody: { data: [orderLive], pagination: { total: 1, page: 1, limit: 20 } },
    shadowBody: { data: [orderLive], pagination: { total: 1, page: 1, limit: 20 } },
    liveStatus: 200,
    shadowStatus: 200,
    diffOps: [],
  },
  {
    method: 'DELETE',
    path: '/v1/orders/ord_9z2x',
    liveBody: { deleted: true },
    shadowBody: { deleted: true },
    liveStatus: 200,
    shadowStatus: 200,
    diffOps: [],
  },
]

// ── Seeding helpers ─────────────────────────────────────────────────

interface SeededGate {
  gate: { id: string; name: string; live_url: string; shadow_url: string; created_at: string }
  requests: { id: string; method: string; path: string; created_at: string }[]
}

/** Seed all three gates. Returns the orders gate with its requests for detail views. */
export async function seedScreenshotData(api: ApiHelper): Promise<{
  ordersGate: SeededGate
  usersGate: SeededGate
  paymentsGate: SeededGate
}> {
  // Create all gates
  const ordersGateData = await api.createGate(gates.orders.name, gates.orders.liveUrl, gates.orders.shadowUrl)
  const usersGateData = await api.createGate(gates.users.name, gates.users.liveUrl, gates.users.shadowUrl)
  const paymentsGateData = await api.createGate(gates.payments.name, gates.payments.liveUrl, gates.payments.shadowUrl)

  // Seed requests for the orders gate (the one we use for detail screenshots)
  const seededRequests = []
  for (const req of ordersRequests) {
    seededRequests.push(
      await api.seedRequest(ordersGateData.id, {
        method: req.method,
        path: req.path,
        liveBody: btoa(JSON.stringify(req.liveBody)),
        shadowBody: btoa(JSON.stringify(req.shadowBody)),
        liveStatus: req.liveStatus,
        shadowStatus: req.shadowStatus,
        diffContent: req.diffOps,
      }),
    )
  }

  return {
    ordersGate: { gate: ordersGateData, requests: seededRequests },
    usersGate: { gate: usersGateData, requests: [] },
    paymentsGate: { gate: paymentsGateData, requests: [] },
  }
}
