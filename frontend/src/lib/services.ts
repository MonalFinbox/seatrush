import { api } from "./api";
import {
  bookingSchema,
  eventSchema,
  holdResponseSchema,
  seatSchema,
  tokenPairSchema,
  userSchema,
  venueRequestSchema,
  venueSchema,
  type AdminLoginForm,
  type CreateEventForm,
  type LoginForm,
  type RegisterForm,
} from "@/types/schemas";
import { z } from "zod";

// ---- Auth ----
export const authApi = {
  async register(body: RegisterForm) {
    const { data } = await api.post("/auth/register", body);
    return tokenPairSchema.parse(data);
  },
  async login(body: LoginForm) {
    const { data } = await api.post("/auth/login", body);
    return tokenPairSchema.parse(data);
  },
  async adminLogin(body: AdminLoginForm) {
    const { data } = await api.post("/auth/admin/login", body);
    return tokenPairSchema.parse(data);
  },
  async activate(paymentMock: Record<string, unknown>) {
    const { data } = await api.post("/auth/organizer/activate", { paymentMock });
    return tokenPairSchema.parse(data);
  },
  async me() {
    const { data } = await api.get("/users/me");
    return userSchema.parse(data);
  },
};

// ---- Venues ----
export const venuesApi = {
  async list(status?: "unclaimed" | "claimed") {
    const { data } = await api.get("/venues", { params: status ? { status } : {} });
    return z.array(venueSchema).parse(data);
  },
  async get(id: string) {
    const { data } = await api.get(`/venues/${id}`);
    return venueSchema.parse(data);
  },
};

// ---- Venue registration requests ----
export const requestsApi = {
  async submit(venueId: string, documentMock: string) {
    const { data } = await api.post(`/venues/${venueId}/registration-requests`, {
      documentMock,
    });
    return venueRequestSchema.parse(data);
  },
  async mine() {
    const { data } = await api.get("/venues/registration-requests/me");
    return z.array(venueRequestSchema).parse(data);
  },
  async adminList(status?: string) {
    const { data } = await api.get("/admin/venue-registration-requests", {
      params: status ? { status } : {},
    });
    return z.array(venueRequestSchema).parse(data);
  },
  async approve(id: string) {
    const { data } = await api.post(`/admin/venue-registration-requests/${id}/approve`);
    return venueRequestSchema.parse(data);
  },
  async reject(id: string, reason: string) {
    const { data } = await api.post(`/admin/venue-registration-requests/${id}/reject`, {
      reason,
    });
    return venueRequestSchema.parse(data);
  },
};

// ---- Events ----
export const eventsApi = {
  async list(status?: string) {
    const { data } = await api.get("/events", { params: status ? { status } : {} });
    return z.array(eventSchema).parse(data);
  },
  async get(id: string) {
    const { data } = await api.get(`/events/${id}`);
    return eventSchema.parse(data);
  },
  async create(body: CreateEventForm) {
    const payload = {
      venueId: body.venueId,
      title: body.title,
      description: body.description || undefined,
      eventDate: new Date(body.eventDate).toISOString(),
    };
    const { data } = await api.post("/events", payload);
    return eventSchema.parse(data);
  },
  async publish(id: string) {
    const { data } = await api.post(`/events/${id}/publish`);
    return eventSchema.parse(data);
  },
  async cancel(id: string) {
    const { data } = await api.post(`/events/${id}/cancel`);
    return eventSchema.parse(data);
  },
};

// ---- Seats ----
export interface SeatInput {
  section: string;
  row: string;
  number: string;
  price: number;
}
export const seatsApi = {
  async map(eventId: string) {
    const { data } = await api.get(`/events/${eventId}/seats`);
    return z.array(seatSchema).parse(data);
  },
  async create(eventId: string, seats: SeatInput[]) {
    const { data } = await api.post(`/events/${eventId}/seats`, { seats });
    return z.object({ created: z.number() }).parse(data);
  },
};

// ---- Holds ----
export const holdsApi = {
  async create(eventId: string, seatIds: string[]) {
    const { data } = await api.post(`/events/${eventId}/holds`, { seatIds });
    return holdResponseSchema.parse(data);
  },
  async release(holdId: string) {
    await api.delete(`/holds/${holdId}`);
  },
};

// ---- Bookings ----
export const bookingsApi = {
  async create(holdId: string, paymentMock: Record<string, unknown>) {
    const { data } = await api.post("/bookings", { holdId, paymentMock });
    return bookingSchema.parse(data);
  },
  async mine() {
    const { data } = await api.get("/bookings");
    return z.array(bookingSchema).parse(data);
  },
  async cancel(id: string) {
    await api.post(`/bookings/${id}/cancel`);
  },
};

// ---- Admin dashboard ----
export const adminApi = {
  async events() {
    const { data } = await api.get("/admin/events");
    return z.array(eventSchema).parse(data);
  },
  async bookings() {
    const { data } = await api.get("/admin/bookings");
    return z.array(bookingSchema).parse(data);
  },
  async users() {
    const { data } = await api.get("/admin/users");
    return z.array(userSchema).parse(data);
  },
};
