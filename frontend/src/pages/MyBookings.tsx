import { useEffect, useState } from "react";
import { toast } from "sonner";
import { bookingsApi } from "@/lib/services";
import { apiError } from "@/lib/api";
import { formatDate, formatINR } from "@/lib/utils";
import type { Booking } from "@/types/schemas";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/layout/StatusBadge";

export default function MyBookings() {
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [loading, setLoading] = useState(true);
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = () => {
    bookingsApi
      .mine()
      .then(setBookings)
      .catch((err) => toast.error(apiError(err)))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const cancel = async (id: string) => {
    setBusyId(id);
    try {
      await bookingsApi.cancel(id);
      toast.success("Booking cancelled");
      load();
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setBusyId(null);
    }
  };

  return (
    <div className="container py-10">
      <h1 className="mb-6 text-3xl font-bold">My bookings</h1>

      {loading ? (
        <Skeleton className="h-40 w-full" />
      ) : bookings.length === 0 ? (
        <p className="text-muted-foreground">You haven't booked anything yet.</p>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {bookings.map((b) => (
            <Card key={b.id}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base">{formatINR(b.totalAmount)}</CardTitle>
                  <StatusBadge status={b.status} />
                </div>
              </CardHeader>
              <CardContent className="space-y-2 text-sm text-muted-foreground">
                <div>{b.seatIds.length} seat(s)</div>
                <div>Booked {formatDate(b.createdAt)}</div>
                {b.status === "confirmed" && (
                  <Button
                    variant="destructive"
                    size="sm"
                    disabled={busyId === b.id}
                    onClick={() => cancel(b.id)}
                  >
                    {busyId === b.id ? "Cancelling…" : "Cancel booking"}
                  </Button>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
