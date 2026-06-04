import { Navigate, Route, Routes } from 'react-router-dom'
import { AdminLayout } from '../layouts/AdminLayout'
import { RequireAdminAuth } from './RequireAdminAuth'
import { AuctionPublishPage } from './pages/AuctionPublishPage'
import { LoginPage } from './pages/LoginPage'
import { AdminRegisterPage } from './pages/RegisterPage'
import { OrderDetailPage } from './pages/OrderDetailPage'
import { OrderListPage } from './pages/OrderListPage'
import {
  ProductCreatePage,
  ProductEditPage,
} from './pages/ProductFormPage'
import { ProductDetailPage } from './pages/ProductDetailPage'
import { DashboardPage } from './pages/DashboardPage'
import { ProductListPage } from './pages/ProductListPage'

export function AdminRoutes() {
  return (
    <Routes>
      <Route path="login" element={<LoginPage />} />
      <Route path="register" element={<AdminRegisterPage />} />
      <Route
        element={
          <RequireAdminAuth>
            <AdminLayout />
          </RequireAdminAuth>
        }
      >
        <Route index element={<DashboardPage />} />
        <Route path="products" element={<ProductListPage />} />
        <Route path="products/new" element={<ProductCreatePage />} />
        <Route path="products/:id" element={<ProductDetailPage />} />
        <Route path="products/:id/edit" element={<ProductEditPage />} />
        <Route path="products/:id/auction" element={<AuctionPublishPage />} />
        <Route path="orders" element={<OrderListPage />} />
        <Route path="orders/:id" element={<OrderDetailPage />} />
      </Route>
    </Routes>
  )
}
