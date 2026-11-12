import { CalendarIcon, XIcon } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { addDays, fmtISO, friendlyDay, nextMonday, parseDay } from "../lib/dates";

interface Props {
  /** YYYY-MM-DD, or "" for none. */
  value: string;
  onChange: (date: string) => void;
  today: string;
  label: string;
  placeholder?: string;
  /** Show a clear button when a date is set. */
  clearable?: boolean;
}

/** A date button that opens a calendar, with Today / Tomorrow / Next Monday shortcuts. */
export function DatePicker({ value, onChange, today, label, placeholder = "Pick a date", clearable }: Props) {
  const [open, setOpen] = useState(false);
  const pick = (d: string) => { setOpen(false); onChange(d); };
  return (
    <span className="inline-flex items-center gap-1">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button variant="outline" size="sm" aria-label={label}>
            <CalendarIcon data-icon="inline-start" />
            {value ? friendlyDay(value, today) : placeholder}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="start" onClick={(e) => e.stopPropagation()}>
          <div className="flex gap-1 border-b p-2">
            <Button variant="ghost" size="xs" onClick={() => pick(today)}>Today</Button>
            <Button variant="ghost" size="xs" onClick={() => pick(addDays(today, 1))}>Tomorrow</Button>
            <Button variant="ghost" size="xs" onClick={() => pick(nextMonday(today))}>Next Monday</Button>
          </div>
          <Calendar
            mode="single"
            required
            selected={value ? parseDay(value) : undefined}
            defaultMonth={value ? parseDay(value) : parseDay(today || fmtISO(new Date()))}
            onSelect={(d) => d && pick(fmtISO(d))}
          />
        </PopoverContent>
      </Popover>
      {clearable && value && (
        <Button variant="ghost" size="icon-xs" aria-label={`Clear ${label.toLowerCase()}`} onClick={() => onChange("")}>
          <XIcon />
        </Button>
      )}
    </span>
  );
}
