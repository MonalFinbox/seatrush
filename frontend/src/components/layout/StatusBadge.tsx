import { Badge } from "@/components/ui/badge";

const variantMap: Record<string, "default" | "secondary" | "destructive" | "success" | "warning" | "outline"> = {
  // events
  draft: "secondary",
  published: "success",
  cancelled: "destructive",
  completed: "outline",
  // requests
  pending: "warning",
  approved: "success",
  rejected: "destructive",
  // venues
  unclaimed: "secondary",
  claimed: "success",
  // users
  active: "success",
  pending_payment: "warning",
  // bookings
  confirmed: "success",
  // seats
  available: "success",
  held: "warning",
  booked: "destructive",
};

export function StatusBadge({ status }: { status: string }) {
  return <Badge variant={variantMap[status] ?? "default"}>{status.replace("_", " ")}</Badge>;
}
