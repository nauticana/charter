# Temporal Package

`Period` evaluates a `Validity`: `Contains(time)`, `Ordered()`, and `Overlaps(Period)`, failing closed on unparsable dates. `EffectiveAt` treats an absent validity as open-ended; `Window` evaluates date-time intervals such as an approval's issue and expiry.
