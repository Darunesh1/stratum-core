// src/main.tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { createRootRoute, createRoute, createRouter, RouterProvider } from '@tanstack/react-router'
import { Root } from './routes/__root'
import { Index } from './routes/index'
import { Ingest } from './routes/ingest'
import { Sql } from './routes/sql'
import { Docs } from './routes/docs'
import './index.css'

// 1. Programmatic Route Tree Setup
const rootRoute = createRootRoute({
  component: Root,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: Index,
})

const ingestRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/ingest',
  component: Ingest,
})

const sqlRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/sql',
  component: Sql,
})

const docsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/docs',
  component: Docs,
})

const routeTree = rootRoute.addChildren([indexRoute, ingestRoute, sqlRoute, docsRoute])

// 2. Initialize Router instance
const router = createRouter({ routeTree })

// 3. Register router type for type safety
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

// 4. Render App
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>,
)
