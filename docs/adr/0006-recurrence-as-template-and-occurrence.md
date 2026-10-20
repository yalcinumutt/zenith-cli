# Recurrence: a Repeater template producing ordinary Task occurrences

Today's single-row `recurring` column (`daily|weekly|monthly|none` rolled forward in place) is replaced with a Repeater entity holding the rule and template fields, where each occurrence is an ordinary Task row pointing back at it. Completing an occurrence is a normal Task completion; the core generates the next occurrence. This was chosen over extending the single-row model because the single row has no history (a completed occurrence has nowhere to go but be overwritten) and no safe sync story (two devices completing the "same" row both roll it forward independently, with no way to tell which generated occurrence is whose).

The rule format itself is deliberately small for v1 (every N days/weeks on given weekdays/months on a given day-of-month) rather than full RRULE (RFC 5545) — the format is cheap to widen later; the template/occurrence split is the hard-to-reverse part, done now.
