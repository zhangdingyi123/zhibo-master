import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AdminRoutes } from './admin/AdminRoutes'
import { LiveRedirectPage } from './user/pages/LiveRedirectPage'
import { UserRoutes } from './user/UserRoutes'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Navigate to="/app" replace />} />
        <Route path="/live" element={<LiveRedirectPage />} />
        <Route path="/admin/*" element={<AdminRoutes />} />
        <Route path="/*" element={<UserRoutes />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
