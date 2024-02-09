package taskmanager

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/capnspacehook/taskmaster"
)

const (
	// Task trigger types that are supported
	BootTask     = "boot"
	LogonTask    = "logon"
	IdleTask     = "idle"
	CreationTask = "creation"
	TimeTask     = "datetime"
	DailyTask    = "time_of_day"
	WeeklyTask   = "time_of_week"
	MonthlyTask  = "time_of_month"
)

// Schemas for JSON output
/*
Some of these types will be similar to the types in the taskmaster library,
but these are modified slightly for more user friendly input and output of
task information
*/
// Information about a task folder
type FolderInfo struct {
	Path string `json:"path"`
}

// Information about a task
type TaskInfo struct {
	// Name of the task
	Name string `json:"name"`
	// Scheduler path (folder)
	Path string `json:"path"`
	// True if the task is enabled, false if not
	Enabled bool `json:"enabled"`
	// Last run time as a local time expressed as an RFC3339 timestamp
	LastRun string `json:"lastRun"`
	// Next run time as a local time expressed as an RFC3339 timestamp
	NextRun string `json:"nextRun"`
	// The status of the task (running, ready...)
	Status string `json:"status"`
	// The execution action for the task
	Actions []string `json:"execute_actions"`
}

/*
Task Definition
Not all of the fields are covered here. This struct is for fields that
the user will be allowed to change. Other values will remain at their
defaults.
*/
type TaskDefinition struct {
	AllowDemandStart          bool      `json:"allow_demand_start"`
	AllowHardTerminate        bool      `json:"allow_hard_terminate"`
	DontStartOnBatteries      bool      `json:"dont_start_on_batteries"`
	Enabled                   bool      `json:"enabled"`
	Hidden                    bool      `json:"hidden"`
	IdleDurationHours         uint      `json:"idle_duration_hours"`
	IdleDurationMinutes       uint      `json:"idle_duration_minutes"`
	IdleDurationSeconds       uint      `json:"idle_duration_seconds"`
	WaitTimeoutHours          uint      `json:"wait_timeout_hours"`
	WaitTimeoutMinutes        uint      `json:"wait_timeout_minutes"`
	WaitTimeoutSeconds        uint      `json:"wait_timeout_seconds"`
	Priority                  uint      `json:"priority"`
	RestartCount              uint      `json:"restart_count"`
	RestartOnIdle             bool      `json:"restart_on_idle"`
	RunOnlyIfIdle             bool      `json:"run_only_if_idle"`
	RunOnlyIfNetworkAvailable bool      `json:"run_only_if_network_available"`
	StartWhenAvailable        bool      `json:"start_when_available"`
	StopIfGoingOnBatteries    bool      `json:"stop_if_going_on_batteries"`
	StopOnIdleEnd             bool      `json:"stop_on_idle_end"`
	TimeLimitHours            uint      `json:"time_limit_hours"`
	TimeLimitMinutes          uint      `json:"time_limit_minutes"`
	TimeLimitSeconds          uint      `json:"time_limit_seconds"`
	WakeToRun                 bool      `json:"wake_to_run"`
	Triggers                  []Trigger `json:"triggers"`
}

type Trigger struct {
	/*
		A condition to trigger this task on. One of:
		boot
		logon
		idle
		creation
		time_of_day
		time_of_week
		time_of_month
	*/
	TriggerOn string `json:"trigger_on"`
	Enabled   bool   `json:"enabled"`
	/*
		Number of seconds to delay executing the task after the trigger condition
		(or a random delay for time_of_day, time_of_week, and time_of_month tasks)
	*/
	Delay uint `json:"delay"`
	// Specifies the user the task will run as for a logon task (blank for current, * for all, name for a specific user)
	User string `json:"user"`
	// Number of seconds the task is allowed to run
	TimeLimit uint `json:"time_limit"`
	// Time specified as %H:%M (24-hour clock) or RFC3339 datetime (datetime is for trigger_on: datetime)
	StartTime string `json:"start_time"`
	// Time specified as %H:%M (24-hour clock)
	EndTime string `json:"end_time"`
	// Populating these fields will depend on the value of TriggerOn

	// Currently only every day (1) or every other day (2) is supported
	DayInterval uint `json:"day_interval,omitempty"`
	// A comma separated list of days. Days are numbered 1 - 7 starting on Sunday, * means every day
	DaysOfWeek string `json:"days_of_week,omitempty"`
	// A comma separated list of days. Days are numbered 1 - 31, * means every day, "last" means the last day of the month
	DaysOfMonth string `json:"days_of_month,omitempty"`
	// A comma separated list of months. Months are numbered 1 - 12, starting in January. * means every month
	MonthsOfYear         string `json:"months_of_year,omitempty"`
	RunOnLastWeekOfMonth bool   `json:"run_on_last_week_of_month,omitempty"`
}

func (t *Trigger) MarshalJSON() ([]byte, error) {
	type TriggerJSON Trigger

	switch t.TriggerOn {
	case DailyTask:
		return json.Marshal(&struct {
			*TriggerJSON
			DayInterval uint `json:"day_interval"`
		}{
			TriggerJSON: (*TriggerJSON)(t),
			DayInterval: t.DayInterval,
		})
	case WeeklyTask:
		return json.Marshal(&struct {
			*TriggerJSON
			DaysOfWeek string `json:"days_of_week"`
		}{
			TriggerJSON: (*TriggerJSON)(t),
			DaysOfWeek:  t.DaysOfWeek,
		})
	case MonthlyTask:
		return json.Marshal(&struct {
			*TriggerJSON
			DaysOfMonth          string `json:"days_of_month"`
			MonthsOfYear         string `json:"months_of_year"`
			RunOnLastWeekOfMonth bool   `json:"run_on_last_week_of_month"`
		}{
			TriggerJSON:          (*TriggerJSON)(t),
			DaysOfMonth:          t.DaysOfMonth,
			MonthsOfYear:         t.MonthsOfYear,
			RunOnLastWeekOfMonth: t.RunOnLastWeekOfMonth,
		})
	default:
		return json.Marshal(&struct {
			*TriggerJSON
		}{
			TriggerJSON: (*TriggerJSON)(t),
		})
	}
}

func (t *Trigger) UnmarshalJSON(data []byte) error {
	type TriggerJSON Trigger
	intermediate := &struct {
		*TriggerJSON
	}{
		TriggerJSON: (*TriggerJSON)(t),
	}
	if err := json.Unmarshal(data, &intermediate); err != nil {
		return err
	}
	return nil
}

func removeDuplicates[T comparable](slice []T) []T {
	allElements := make(map[T]bool)
	newSlice := []T{}

	for _, element := range slice {
		if _, value := allElements[element]; !value {
			allElements[element] = true
			newSlice = append(newSlice, element)
		}
	}
	return newSlice
}

// Convert a comma separated list of days of the week into something the taskmaster library will understand
func (t *Trigger) ConvertDaysOfWeek() (taskmaster.DayOfWeek, error) {
	var representation taskmaster.DayOfWeek = 0

	defaultErr := fmt.Errorf("%s is not a valid list of week days", t.DaysOfWeek)

	// Get rid of any spaces
	requestedDaysStr := strings.ReplaceAll(t.DaysOfWeek, " ", "")
	requestedDays := strings.Split(requestedDaysStr, ",")
	requestedDays = removeDuplicates(requestedDays)

	for _, day := range requestedDays {
		dayNum, err := strconv.Atoi(day)
		if err != nil {
			return representation, defaultErr
		}
		if dayNum <= 0 || dayNum > 7 {
			return representation, defaultErr
		}
		representation = representation | taskmaster.DayOfWeek(1<<(dayNum-1))
	}

	return representation, nil
}

/*
Convert the representation of days of the week from the taskmaster library
into something that is easier to work with when interacting with the user (a
comma separated list of days).

The taskmaster library has a function that turns the representation into a string
but it outputs the names of the days which are a bit cumbersome to work with.
*/
func (t *Trigger) DaysOfWeekFromTrigger(days taskmaster.DayOfWeek) error {
	// We can use the logic that the taskmaster library uses but change the string that gets output
	var tempBuf []string

	if days == taskmaster.AllDays {
		t.DaysOfWeek = "*"
		return nil
	}

	if days == 0 || days > taskmaster.AllDays {
		return fmt.Errorf("invalid days of the week")
	}

	if taskmaster.Sunday&days == taskmaster.Sunday {
		tempBuf = append(tempBuf, "1")
	}
	if taskmaster.Monday&days == taskmaster.Monday {
		tempBuf = append(tempBuf, "2")
	}
	if taskmaster.Tuesday&days == taskmaster.Tuesday {
		tempBuf = append(tempBuf, "3")
	}
	if taskmaster.Wednesday&days == taskmaster.Wednesday {
		tempBuf = append(tempBuf, "4")
	}
	if taskmaster.Thursday&days == taskmaster.Thursday {
		tempBuf = append(tempBuf, "5")
	}
	if taskmaster.Friday&days == taskmaster.Friday {
		tempBuf = append(tempBuf, "6")
	}
	if taskmaster.Saturday&days == taskmaster.Saturday {
		tempBuf = append(tempBuf, "7")
	}

	t.DaysOfWeek = strings.Join(tempBuf, ",")

	return nil
}

/*
Convert a taskmaster representation of days of the month into a
comma separated list of months.
*/
func (t *Trigger) DaysOfMonthFromTrigger(days taskmaster.DayOfMonth) error {
	if days == 0 || days > taskmaster.AllDaysOfMonth {
		return fmt.Errorf("invalid days of the month")
	}
	if days == taskmaster.AllDaysOfMonth {
		t.DaysOfMonth = "*"
		return nil
	}

	// We can use the logic that the taskmaster library uses but change the string that gets output
	var tempBuf []string
	for i, j := taskmaster.DayOfMonth(1), uint(1); i < taskmaster.LastDayOfMonth; i, j = (1<<j+1)-1, j+1 {
		if days&i == i {
			tempBuf = append(tempBuf, strconv.Itoa(int(j)))
		}
	}

	if days&taskmaster.LastDayOfMonth == taskmaster.LastDayOfMonth {
		tempBuf = append(tempBuf, "last")
	}

	t.DaysOfMonth = strings.Join(tempBuf, ",")
	return nil
}

// Converts a taskmaster representation of months to a comma separated list of month numbers
func (t *Trigger) MonthsOfYearFromTrigger(months taskmaster.Month) error {
	// We can use the logic that the taskmaster library uses but change the string that gets output
	var tempBuf []string

	if months == 0 || months > taskmaster.AllMonths {
		return fmt.Errorf("invalid months of the year")
	}
	if months == taskmaster.AllMonths {
		t.DaysOfMonth = "*"
		return nil
	}
	if taskmaster.January&months == taskmaster.January {
		tempBuf = append(tempBuf, "1")
	}
	if taskmaster.February&months == taskmaster.February {
		tempBuf = append(tempBuf, "2")
	}
	if taskmaster.March&months == taskmaster.March {
		tempBuf = append(tempBuf, "3")
	}
	if taskmaster.April&months == taskmaster.April {
		tempBuf = append(tempBuf, "4")
	}
	if taskmaster.May&months == taskmaster.May {
		tempBuf = append(tempBuf, "5")
	}
	if taskmaster.June&months == taskmaster.June {
		tempBuf = append(tempBuf, "6")
	}
	if taskmaster.July&months == taskmaster.July {
		tempBuf = append(tempBuf, "7")
	}
	if taskmaster.August&months == taskmaster.August {
		tempBuf = append(tempBuf, "8")
	}
	if taskmaster.September&months == taskmaster.September {
		tempBuf = append(tempBuf, "9")
	}
	if taskmaster.October&months == taskmaster.October {
		tempBuf = append(tempBuf, "10")
	}
	if taskmaster.November&months == taskmaster.November {
		tempBuf = append(tempBuf, "11")
	}
	if taskmaster.December&months == taskmaster.December {
		tempBuf = append(tempBuf, "12")
	}

	t.DaysOfMonth = strings.Join(tempBuf, ",")

	return nil
}

// Convert a Trigger's days of month into something the taskmaster library will understand
func (t *Trigger) ConvertDaysOfMonth() (taskmaster.DayOfMonth, error) {
	var representation taskmaster.DayOfMonth = 0

	defaultErr := fmt.Errorf("%s is not a valid list of days of the month", t.DaysOfMonth)

	// Get rid of any spaces
	requestedDaysStr := strings.ReplaceAll(t.DaysOfMonth, " ", "")
	requestedDays := strings.Split(requestedDaysStr, ",")
	requestedDays = removeDuplicates(requestedDays)

	for _, day := range requestedDays {
		if day == "last" {
			representation = representation | taskmaster.LastDayOfMonth
			continue
		}
		dayNum, err := strconv.Atoi(day)
		if err != nil {
			return representation, err
		}
		if dayNum <= 0 || dayNum > 31 {
			return representation, defaultErr
		}
		representation = representation | taskmaster.DayOfMonth(1<<(dayNum-1))
	}

	return representation, nil
}

// Convert a Trigger's list of months into something the taskmaster library will understand
func (t *Trigger) ConvertMonths() (taskmaster.Month, error) {
	var representation taskmaster.Month = 0

	defaultErr := fmt.Errorf("%s is not a valid list of months", t.MonthsOfYear)

	// Get rid of any spaces
	requestedMonthsStr := strings.ReplaceAll(t.MonthsOfYear, " ", "")
	requestedMonths := strings.Split(requestedMonthsStr, ",")
	requestedMonths = removeDuplicates(requestedMonths)

	for _, month := range requestedMonths {
		monthNum, err := strconv.Atoi(month)
		if err != nil {
			return representation, defaultErr
		}
		if monthNum <= 0 || monthNum > 12 {
			return representation, defaultErr
		}
		representation = representation | taskmaster.Month(1<<(monthNum-1))
	}

	return representation, nil
}
