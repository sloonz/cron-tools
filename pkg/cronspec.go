package crontools

import (
	"fmt"
	"math"
	"math/bits"
	"strconv"
	"strings"
	"time"
)

// Set that can contain all numbers from 0 to 63. It is simply represented by a
// 4-bytes integer
//
// 0 is in the set if bit 1 (less significant) is set
// 1 is in the set if bit 2 is set
// ...
// 63 is in the set if bit 64 (most significant) is set
type bitSet uint64

func (b bitSet) Empty() bool {
	return b == 0
}

func (b bitSet) Highest() int {
	return 63 - bits.LeadingZeros64(uint64(b))
}

func (b bitSet) Lowest() int {
	return bits.TrailingZeros64(uint64(b))
}

// Find the lowest element m in the set such that m > n.
//
// If n is the highest element in the set, returns the lowest element in
// the set, to mimic integer arithmetic where INT_MAX+1 = INT_MIN
//
// Undefined result for the empty set
func (b bitSet) NextHigher(n int) int {
	highSet := bitSet(b & (math.MaxUint64 ^ ((1 << (n + 1)) - 1)))
	if highSet.Empty() {
		return b.Lowest()
	}
	return highSet.Lowest()
}

// Find the highest element m in the set such that m < n
//
// If n is the lowest element in the set, returns the highest element in
// the set, to mimic integer arithmetic where INT_MIN-1 = INT_MAX
//
// Undefined result for the empty set
func (b bitSet) NextLower(n int) int {
	lowSet := bitSet(b & ((1 << n) - 1))
	if lowSet.Empty() {
		return b.Highest()
	}
	return lowSet.Highest()
}

func (b bitSet) Add(n int) bitSet {
	return b | (1 << n)
}

func (b bitSet) Has(n int) bool {
	return (b & (1 << n)) != 0
}

type cronSpec struct {
	minuteSet   bitSet
	hourSet     bitSet
	monthSet    bitSet
	dowSet      bitSet
	domSet      bitSet
	dayWildcard bool
}

func (s *cronSpec) MinuteSet() bitSet {
	return s.minuteSet
}

func (s *cronSpec) HourSet() bitSet {
	return s.hourSet
}

func (s *cronSpec) MonthSet() bitSet {
	return s.monthSet
}

func (s *cronSpec) DaySet(year, month int) bitSet {
	var res bitSet
	t := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	for ; int(t.Month()) == month; t = t.AddDate(0, 0, 1) {
		domCheck := s.domSet.Has(t.Day())
		dowCheck := s.dowSet.Has(int(t.Weekday()))
		if s.dayWildcard {
			if domCheck && dowCheck {
				res = res.Add(t.Day())
			}
		} else {
			if domCheck || dowCheck {
				res = res.Add(t.Day())
			}
		}
	}
	return res
}

func (s *cronSpec) NextDate(t time.Time) time.Time {
	oldMinute := t.Minute()
	newMinute := s.MinuteSet().NextHigher(oldMinute)
	carry := newMinute <= oldMinute

	oldHour := t.Hour()
	newHour := oldHour
	if carry || !s.HourSet().Has(oldHour) {
		newHour = s.HourSet().NextHigher(oldHour)
		newMinute = s.MinuteSet().Lowest()
		carry = newHour <= oldHour
	} else {
		carry = false
	}

	oldDay := t.Day()
	newDay := oldDay
	if carry || !s.DaySet(t.Year(), int(t.Month())).Has(oldDay) {
		newDay = s.DaySet(t.Year(), int(t.Month())).NextHigher(oldDay)
		newHour = s.HourSet().Lowest()
		newMinute = s.MinuteSet().Lowest()
		carry = newDay <= oldDay
	} else {
		carry = false
	}

	oldMonth := int(t.Month())
	newMonth := oldMonth
	year := t.Year()
	if carry || !s.MonthSet().Has(oldMonth) {
		newMonth = s.MonthSet().NextHigher(oldMonth)
		if newMonth <= oldMonth {
			year++
		}
		newDay = s.DaySet(year, newMonth).Lowest()
		newHour = s.HourSet().Lowest()
		newMinute = s.MinuteSet().Lowest()
	}

	return time.Date(year, time.Month(newMonth), newDay, newHour, newMinute, 0, 0, t.Location())
}

func (s *cronSpec) PreviousDate(t time.Time) time.Time {
	oldMinute := t.Minute()
	newMinute := s.MinuteSet().NextLower(oldMinute)
	carry := newMinute >= oldMinute

	oldHour := t.Hour()
	newHour := oldHour
	if carry || !s.HourSet().Has(oldHour) {
		newHour = s.HourSet().NextLower(oldHour)
		newMinute = s.MinuteSet().Highest()
		carry = newHour >= oldHour
	} else {
		carry = false
	}

	oldDay := t.Day()
	newDay := oldDay
	if carry || !s.DaySet(t.Year(), int(t.Month())).Has(oldDay) {
		newDay = s.DaySet(t.Year(), int(t.Month())).NextLower(oldDay)
		newHour = s.HourSet().Highest()
		newMinute = s.MinuteSet().Highest()
		carry = newDay >= oldDay
	} else {
		carry = false
	}

	oldMonth := int(t.Month())
	newMonth := oldMonth
	year := t.Year()
	if carry || !s.MonthSet().Has(oldMonth) {
		newMonth = s.MonthSet().NextLower(oldMonth)
		if newMonth >= oldMonth {
			year--
		}
		newDay = s.DaySet(year, newMonth).Highest()
		newHour = s.HourSet().Highest()
		newMinute = s.MinuteSet().Highest()
	}

	return time.Date(year, time.Month(newMonth), newDay, newHour, newMinute, 0, 0, t.Location())
}

func parseSpecPart(p string, low, high, modulo int, names map[string]int) (bitSet, error) {
	var set bitSet
	var err error

	// Handle a,b,c
	subSets := strings.Split(p, ",")
	if len(subSets) > 1 {
		for _, subPart := range subSets {
			subSet, err := parseSpecPart(subPart, low, high, modulo, names)
			if err != nil {
				return bitSet(0), err
			}
			set |= subSet
		}
		return set, nil
	}

	// Handle names
	if names != nil && len(p) >= 3 {
		n, ok := names[strings.ToLower(p)[0:3]]
		if ok {
			return set.Add(n), nil
		}
	}

	// p/step
	step := 1
	if strings.Contains(p, "/") {
		parts := strings.Split(p, "/")
		if len(parts) != 2 {
			return bitSet(0), fmt.Errorf("invalid step: %s", p)
		}
		step, err = strconv.Atoi(parts[1])
		p = parts[0]
		if err != nil || step < 1 {
			return bitSet(0), fmt.Errorf("invalid step: %s", p)
		}
	}

	if strings.Contains(p, "-") {
		// Range element (a-b)
		parts := strings.Split(p, "-")
		if len(parts) != 2 {
			return bitSet(0), fmt.Errorf("invalid range: %s", p)
		}

		a, err := strconv.Atoi(parts[0])
		if err != nil {
			return bitSet(0), fmt.Errorf("invalid range: %s", p)
		}

		b, err := strconv.Atoi(parts[1])
		if err != nil {
			return bitSet(0), fmt.Errorf("invalid range: %s", p)
		}

		if b < a || a < low || b > high {
			return bitSet(0), fmt.Errorf("invalid range: %s", p)
		}

		low = a
		high = b
	} else if p != "*" {
		// Singleton element
		n, err := strconv.Atoi(p)
		if err != nil {
			return bitSet(0), fmt.Errorf("invalid element: %s", p)
		}
		if n < low || n > high {
			return bitSet(0), fmt.Errorf("invalid element: %s", p)
		}
		low = n
		high = n
	}

	for n := low; n <= high; n += step {
		set = set.Add(n % modulo)
	}

	return set, nil
}

func ParseCronTimeSpecification(s string) (*cronSpec, error) {
	switch s {
	case "@yearly":
		fallthrough
	case "@annually":
		return ParseCronTimeSpecification("0 0 1 1 *")
	case "@monthly":
		return ParseCronTimeSpecification("0 0 1 * *")
	case "@weekly":
		return ParseCronTimeSpecification("0 0 * * 0")
	case "@daily":
		return ParseCronTimeSpecification("0 0 * * *")
	case "@hourly":
		return ParseCronTimeSpecification("0 * * * *")
	}

	dayNames := map[string]int{
		"sun": 0,
		"mon": 1,
		"tue": 2,
		"wed": 3,
		"thu": 4,
		"fri": 5,
		"sat": 6,
	}

	monthNames := map[string]int{
		"jan": 1,
		"feb": 2,
		"mar": 3,
		"apr": 4,
		"may": 5,
		"jun": 6,
		"jul": 7,
		"aug": 8,
		"sep": 9,
		"oct": 10,
		"nov": 11,
		"dec": 12,
	}

	parts := strings.Split(s, " ")
	if len(parts) != 5 {
		return nil, fmt.Errorf("cannot parse %s: invalid parts count", s)
	}

	minuteSpec, err := parseSpecPart(parts[0], 0, 59, 60, nil)
	if err != nil {
		return nil, err
	}

	hourSpec, err := parseSpecPart(parts[1], 0, 23, 24, nil)
	if err != nil {
		return nil, err
	}

	domSpec, err := parseSpecPart(parts[2], 1, 31, 32, nil)
	if err != nil {
		return nil, err
	}

	monthSpec, err := parseSpecPart(parts[3], 1, 12, 13, monthNames)
	if err != nil {
		return nil, err
	}

	dowSpec, err := parseSpecPart(parts[4], 0, 7, 7, dayNames)
	if err != nil {
		return nil, err
	}

	return &cronSpec{
		minuteSet:   minuteSpec,
		hourSet:     hourSpec,
		domSet:      domSpec,
		monthSet:    monthSpec,
		dowSet:      dowSpec,
		dayWildcard: parts[2] == "*" || parts[4] == "*",
	}, nil
}
