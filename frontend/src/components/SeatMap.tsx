import { useMemo } from "react";
import { cn, formatINR } from "@/lib/utils";
import type { Seat } from "@/types/schemas";

interface SeatMapProps {
  seats: Seat[];
  selected: Set<string>;
  onToggle: (seat: Seat) => void;
  interactive: boolean;
}

/** Groups seats into section -> row -> seats for a readable layout. */
function group(seats: Seat[]) {
  const sections = new Map<string, Map<string, Seat[]>>();
  for (const s of seats) {
    if (!sections.has(s.section)) sections.set(s.section, new Map());
    const rows = sections.get(s.section)!;
    if (!rows.has(s.row)) rows.set(s.row, []);
    rows.get(s.row)!.push(s);
  }
  return sections;
}

export function SeatMap({ seats, selected, onToggle, interactive }: SeatMapProps) {
  const sections = useMemo(() => group(seats), [seats]);

  if (seats.length === 0) {
    return <p className="text-sm text-muted-foreground">No seat map defined yet.</p>;
  }

  return (
    <div className="space-y-6">
      {[...sections.entries()].map(([section, rows]) => (
        <div key={section}>
          <h4 className="mb-2 text-sm font-semibold text-muted-foreground">Section {section}</h4>
          <div className="space-y-1.5">
            {[...rows.entries()].map(([row, rowSeats]) => (
              <div key={row} className="flex items-center gap-1.5">
                <span className="w-6 text-xs text-muted-foreground">{row}</span>
                {rowSeats
                  .sort((a, b) => a.number.localeCompare(b.number, undefined, { numeric: true }))
                  .map((seat) => {
                    const isSelected = selected.has(seat.id);
                    const clickable = interactive && (seat.status === "available" || isSelected);
                    return (
                      <button
                        key={seat.id}
                        type="button"
                        disabled={!clickable}
                        onClick={() => onToggle(seat)}
                        title={`${seat.section}${seat.row}-${seat.number} · ${formatINR(seat.price)} · ${seat.status}`}
                        className={cn(
                          "flex h-8 w-8 items-center justify-center rounded text-[10px] font-medium transition-colors",
                          seat.status === "available" && !isSelected &&
                            "bg-emerald-600/20 text-emerald-300 hover:bg-emerald-600/40",
                          isSelected && "bg-primary text-primary-foreground ring-2 ring-primary",
                          seat.status === "held" && !isSelected &&
                            "bg-amber-500/30 text-amber-200 cursor-not-allowed",
                          seat.status === "booked" &&
                            "bg-destructive/30 text-red-300 cursor-not-allowed"
                        )}
                      >
                        {seat.number}
                      </button>
                    );
                  })}
              </div>
            ))}
          </div>
        </div>
      ))}

      <div className="flex flex-wrap gap-4 pt-2 text-xs text-muted-foreground">
        <Legend className="bg-emerald-600/30" label="Available" />
        <Legend className="bg-primary" label="Selected" />
        <Legend className="bg-amber-500/40" label="Held" />
        <Legend className="bg-destructive/40" label="Booked" />
      </div>
    </div>
  );
}

function Legend({ className, label }: { className: string; label: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span className={cn("h-3 w-3 rounded", className)} /> {label}
    </span>
  );
}
