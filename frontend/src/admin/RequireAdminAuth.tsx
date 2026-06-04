import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { isAdminLoggedIn } from '../auth/adminAuth'

export function RequireAdminAuth({ children }: { children: ReactNode }) {
  const location = useLocation()
  if (!isAdminLoggedIn()) {
    return <Navigate to="/admin/login" state={{ from: location }} replace />
  }
  return children
}
