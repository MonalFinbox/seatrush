import { z } from "zod";

/**
 * Schemas mirror the Go backend's JSON shapes (camelCase). Nullable columns
 * use .nullable(); fields the API may omit use .optional().
 */

export const userSchema = z.object({
  id: z.string(),
  email: z.string(),
  role: z.enum(["admin", "organizer", "attendee"]),
  status: z.enum(["active", "pending_payment"]),
  createdAt: z.string(),
  updatedAt: z.string(),
});
export type User = z.infer<typeof userSchema>;

export const tokenPairSchema = z.object({
  user: userSchema,
  accessToken: z.string(),
  refreshToken: z.string(),
});
export type TokenPair = z.infer<typeof tokenPairSchema>;

export const venueSchema = z.object({
  id: z.string(),
  name: z.string(),
  address: z.string(),
  city: z.string(),
  capacity: z.number(),
  claimStatus: z.enum(["unclaimed", "claimed"]),
  organizerId: z.string().nullable(),
  createdAt: z.string(),
  updatedAt: z.string(),
});
export type Venue = z.infer<typeof venueSchema>;

export const venueRequestSchema = z.object({
  id: z.string(),
  venueId: z.string(),
  organizerId: z.string(),
  documentMock: z.string(),
  status: z.enum(["pending", "approved", "rejected"]),
  reviewedBy: z.string().nullable(),
  reviewedAt: z.string().nullable(),
  rejectionReason: z.string().nullable(),
  createdAt: z.string(),
  updatedAt: z.string(),
});
export type VenueRequest = z.infer<typeof venueRequestSchema>;

export const eventSchema = z.object({
  id: z.string(),
  venueId: z.string(),
  organizerId: z.string(),
  title: z.string(),
  description: z.string().nullable(),
  eventDate: z.string(),
  status: z.enum(["draft", "published", "cancelled", "completed"]),
  createdAt: z.string(),
  updatedAt: z.string(),
});
export type Event = z.infer<typeof eventSchema>;

export const seatStatus = z.enum(["available", "held", "booked"]);
export type SeatStatus = z.infer<typeof seatStatus>;

export const seatSchema = z.object({
  id: z.string(),
  eventId: z.string(),
  section: z.string(),
  row: z.string(),
  number: z.string(),
  price: z.number(),
  status: seatStatus,
  createdAt: z.string(),
  updatedAt: z.string(),
});
export type Seat = z.infer<typeof seatSchema>;

export const bookingSchema = z.object({
  id: z.string(),
  userId: z.string(),
  eventId: z.string(),
  status: z.enum(["confirmed", "cancelled"]),
  totalAmount: z.number(),
  seatIds: z.array(z.string()).optional().default([]),
  createdAt: z.string(),
  updatedAt: z.string(),
});
export type Booking = z.infer<typeof bookingSchema>;

export const holdResponseSchema = z.object({
  holdId: z.string(),
  seatIds: z.array(z.string()),
  expiresAt: z.string(),
});
export type HoldResponse = z.infer<typeof holdResponseSchema>;

// ---- WebSocket message ----
export const wsEventSchema = z.object({
  type: z.enum(["seat.held", "seat.released", "seat.booked"]),
  seatId: z.string(),
  timestamp: z.string(),
});
export type WsEvent = z.infer<typeof wsEventSchema>;

// ---- Form schemas ----
export const loginFormSchema = z.object({
  email: z.string().email("Enter a valid email"),
  password: z.string().min(1, "Password is required"),
});
export type LoginForm = z.infer<typeof loginFormSchema>;

export const registerFormSchema = z.object({
  email: z.string().email("Enter a valid email"),
  password: z.string().min(8, "At least 8 characters"),
  role: z.enum(["attendee", "organizer"]),
});
export type RegisterForm = z.infer<typeof registerFormSchema>;

export const adminLoginFormSchema = z.object({
  email: z.string().email("Enter a valid email"),
  password: z.string().min(1, "Password is required"),
  adminAccessKey: z.string().min(1, "Access key is required"),
});
export type AdminLoginForm = z.infer<typeof adminLoginFormSchema>;

export const createEventFormSchema = z.object({
  venueId: z.string().min(1, "Pick a venue"),
  title: z.string().min(1, "Title is required"),
  description: z.string().optional(),
  eventDate: z.string().min(1, "Pick a date"),
});
export type CreateEventForm = z.infer<typeof createEventFormSchema>;
