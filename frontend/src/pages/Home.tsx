import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { CalendarDays, MapPin } from "lucide-react";
import { eventsApi, venuesApi } from "@/lib/services";
import { apiError } from "@/lib/api";
import { formatDate } from "@/lib/utils";
import type { Event, Venue } from "@/types/schemas";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/layout/StatusBadge";
import { toast } from "sonner";

export default function Home() {
  const [events, setEvents] = useState<Event[]>([]);
  const [venues, setVenues] = useState<Record<string, Venue>>({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([eventsApi.list("published"), venuesApi.list()])
      .then(([evs, vns]) => {
        setEvents(evs);
        setVenues(Object.fromEntries(vns.map((v) => [v.id, v])));
      })
      .catch((err) => toast.error(apiError(err)))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="container py-10">
      <div className="mb-8">
        <h1 className="text-3xl font-bold tracking-tight">Live events</h1>
        <p className="text-muted-foreground">Find an event and grab your seats.</p>
      </div>

      {loading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-40 w-full" />
          ))}
        </div>
      ) : events.length === 0 ? (
        <p className="text-muted-foreground">No published events yet. Check back soon.</p>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {events.map((ev) => {
            const venue = venues[ev.venueId];
            return (
              <Link key={ev.id} to={`/events/${ev.id}`}>
                <Card className="h-full transition-colors hover:border-primary">
                  <CardHeader>
                    <div className="flex items-start justify-between gap-2">
                      <CardTitle className="text-lg">{ev.title}</CardTitle>
                      <StatusBadge status={ev.status} />
                    </div>
                    {ev.description && (
                      <CardDescription className="line-clamp-2">{ev.description}</CardDescription>
                    )}
                  </CardHeader>
                  <CardContent className="space-y-2 text-sm text-muted-foreground">
                    <div className="flex items-center gap-2">
                      <CalendarDays className="h-4 w-4" />
                      {formatDate(ev.eventDate)}
                    </div>
                    {venue && (
                      <div className="flex items-center gap-2">
                        <MapPin className="h-4 w-4" />
                        {venue.name}, {venue.city}
                      </div>
                    )}
                  </CardContent>
                </Card>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}
