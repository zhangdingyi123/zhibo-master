import { Route, Routes } from 'react-router-dom'
import { MobileLayout } from './layouts/MobileLayout'
import { AuctionDetailPage } from './pages/AuctionDetailPage'
import { AuctionListPage } from './pages/AuctionListPage'
import { HistoryPage } from './pages/HistoryPage'
import { LiveRoomPage } from './pages/LiveRoomPage'
import { MyOrdersPage } from './pages/MyOrdersPage'
import { OrderPayPage } from './pages/OrderPayPage'
import { ProfilePage } from './pages/ProfilePage'
import { UserLoginPage } from './pages/LoginPage'
import { UserRegisterPage } from './pages/RegisterPage'
import { ResultPage } from './pages/ResultPage'

export function UserRoutes() {
  return (
    <Routes>
      <Route path="/app/login" element={<UserLoginPage />} />
      <Route path="/app/register" element={<UserRegisterPage />} />
      <Route path="/app" element={<MobileLayout />}>
        <Route index element={<AuctionListPage />} />
        <Route path="auction/:sessionId" element={<AuctionDetailPage />} />
        <Route path="live/:roomId" element={<LiveRoomPage />} />
        <Route path="history" element={<HistoryPage />} />
        <Route path="orders" element={<MyOrdersPage />} />
        <Route path="orders/:orderId" element={<OrderPayPage />} />
        <Route path="result/:sessionId" element={<ResultPage />} />
        <Route path="profile" element={<ProfilePage />} />
      </Route>
    </Routes>
  )
}
