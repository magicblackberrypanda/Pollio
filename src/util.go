package main

import (
	"fmt"
	"os"
	"time"
	"errors"
	"strconv"
	"strings"
)

func timestamp() string {
	return time.Now().Format(time.RFC3339)
}

func debugf(format string, v ...any) {
	fmt.Fprintf(os.Stderr, "%s [DEBUG] %s\n", timestamp(), fmt.Sprintf(format, v...))
}

func infof(format string, v ...any) {
	fmt.Fprintf(os.Stderr, "%s [INFO] %s\n", timestamp(), fmt.Sprintf(format, v...))
}

func errorf(format string, v ...any) {
	fmt.Fprintf(os.Stderr, "%s [ERROR] %s\n", timestamp(), fmt.Sprintf(format, v...))
	os.Exit(1)
}

func warningf(format string, v ...any) {
	fmt.Fprintf(os.Stderr, "%s [WARNING] %s\n", timestamp(), fmt.Sprintf(format, v...))
}

// parseInterval accepts "@1m", "@5h", "@1d", or explicit durations like "30s".
func parseInterval(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty interval")
	}
	if strings.HasPrefix(s, "@") {
		v := strings.TrimPrefix(s, "@")
		if strings.HasSuffix(v, "m") || strings.HasSuffix(v, "h") || strings.HasSuffix(v, "d") {
			unit := v[len(v)-1]
			num := v[:len(v)-1]
			n, err := strconv.Atoi(num)
			if err != nil {
				return 0, err
			}
			switch unit {
			case 'm':
				return time.Duration(n) * time.Minute, nil
			case 'h':
				return time.Duration(n) * time.Hour, nil
			case 'd':
				return time.Duration(n) * 24 * time.Hour, nil
			default:
				return 0, fmt.Errorf("unknown unit: %c", unit)
			}
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * time.Second, nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	if n, err := strconv.Atoi(s); err == nil {
		return time.Duration(n) * time.Second, nil
	}
	return 0, fmt.Errorf("invalid interval: %s", s)
}

func inMaintenanceWindow(mp *MaintenancePeriodConfig, now time.Time) bool {
	if mp == nil {
        return false
    }
	if mp.Repeat == "" && mp.StartingDay == "" && mp.StartingTime == "" && mp.Duration == "" {
		return false
	}

	dur, err := parseInterval(mp.Duration)
	if err != nil || dur <= 0 {
		return false
	}

	st, err := time.Parse("15:04", mp.StartingTime)
	if err != nil {
		return false
	}

	rep := strings.ToLower(mp.Repeat)
	day := strings.ToLower(mp.StartingDay)
	loc := now.Location()

	switch {
	case strings.Contains(rep, "daily"):
		start := time.Date(now.Year(), now.Month(), now.Day(), st.Hour(), st.Minute(), 0, 0, loc)
		// if start is after now, consider previous day's start only if duration could cover it (skip for simplicity)
		return !now.Before(start) && now.Sub(start) < dur

	case strings.Contains(rep, "weekly"):
		wdMap := map[string]time.Weekday{
			"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
			"wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday,
		}
		target, ok := wdMap[day]
		if !ok {
			return false
		}
		daysAgo := (int(now.Weekday()) - int(target) + 7) % 7
		startDate := now.AddDate(0, 0, -daysAgo)
		start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), st.Hour(), st.Minute(), 0, 0, loc)
		if start.After(now) {
			start = start.AddDate(0, 0, -7)
		}
		return !now.Before(start) && now.Sub(start) < dur

	case strings.Contains(rep, "monthly"):
		wdMap := map[string]time.Weekday{
			"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
			"wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday,
		}
		target, ok := wdMap[day]
		if !ok {
			return false
		}
		first := time.Date(now.Year(), now.Month(), 1, st.Hour(), st.Minute(), 0, 0, loc)
		daysUntil := (int(target) - int(first.Weekday()) + 7) % 7
		start := first.AddDate(0, 0, daysUntil)
		for start.AddDate(0, 0, 7).Before(now) || start.AddDate(0, 0, 7).Equal(now) {
			start = start.AddDate(0, 0, 7)
		}
		return !now.Before(start) && now.Sub(start) < dur

	default:
		// fallback: if repeat contains any known token, try weekly logic
		if strings.Contains(rep, "weekly") || strings.Contains(rep, "monthly") || strings.Contains(rep, "daily") {
			mp2 := mp
			if strings.Contains(rep, "weekly") {
				mp2.Repeat = "weekly"
			} else if strings.Contains(rep, "daily") {
				mp2.Repeat = "daily"
			} else {
				mp2.Repeat = "monthly"
			}
			return inMaintenanceWindow(mp2, now)
		}
		return false
	}
}
