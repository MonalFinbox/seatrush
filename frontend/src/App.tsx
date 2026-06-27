import { BrowserRouter, Routes, Route, Outlet } from "react-router-dom";
import { Toaster } from "sonner";
import { AuthProvider } from "@/store/auth";
import { Navbar } from "@/components/layout/Navbar";
import { ProtectedRoute } from "@/components/layout/ProtectedRoute";

import Home from "@/pages/Home";
import Login from "@/pages/Login";
import Register from "@/pages/Register";
import AdminLogin from "@/pages/AdminLogin";
import EventDetail from "@/pages/EventDetail";
import MyBookings from "@/pages/MyBookings";
import OrganizerDashboard from "@/pages/organizer/OrganizerDashboard";
import EventManage from "@/pages/organizer/EventManage";
import AdminDashboard from "@/pages/admin/AdminDashboard";

function Layout() {
  return (
    <div className="min-h-screen">
      <Navbar />
      <main>
        <Outlet />
      </main>
    </div>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route element={<Layout />}>
            {/* Public */}
            <Route path="/" element={<Home />} />
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
            <Route path="/admin/login" element={<AdminLogin />} />
            <Route path="/events/:eventId" element={<EventDetail />} />

            {/* Attendee */}
            <Route element={<ProtectedRoute roles={["attendee"]} />}>
              <Route path="/bookings" element={<MyBookings />} />
            </Route>

            {/* Organizer */}
            <Route element={<ProtectedRoute roles={["organizer"]} />}>
              <Route path="/organizer" element={<OrganizerDashboard />} />
              <Route path="/organizer/events/:eventId" element={<EventManage />} />
            </Route>

            {/* Admin */}
            <Route element={<ProtectedRoute roles={["admin"]} />}>
              <Route path="/admin" element={<AdminDashboard />} />
            </Route>

            <Route path="*" element={<div className="container py-20 text-center">Not found.</div>} />
          </Route>
        </Routes>
      </BrowserRouter>
      <Toaster richColors position="top-right" />
    </AuthProvider>
  );
}
